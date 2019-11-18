package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	stdurl "net/url"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/yhat/scrape"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/sync/errgroup"
)

func main() {
	config, err := NewConfig()
	if err != nil {
		log.Fatal(errors.Wrap(err, "parsing config"))
	}
	app, err := NewApp(config)
	if err != nil {
		log.Fatal(errors.Wrap(err, "initializing app"))
	}
	if err := app.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

// App defines the application's behavior.
type App struct {
	Config
}

// NewApp initializes the application.
func NewApp(conf Config) (*App, error) {
	app := &App{
		Config: conf,
	}
	return app, nil
}

// Run runs the application.
func (app *App) Run(ctx context.Context) error {
	if app.Download {
		return app.download(ctx)
	}
	return app.list(ctx)
}

func (app *App) contentFetcher(ctx context.Context, download string, dc chan Download) func() error {
	return func() error {
		log.Printf("downloading %s", download)

		resp, err := http.Get(download)
		if err != nil {
			panic(err)
		}
		if resp.StatusCode >= http.StatusMultipleChoices {
			return errors.New(download + ": " + resp.Status)
		}
		select {
		case <-ctx.Done():
			return nil
		case dc <- Download{Content: resp.Body, Location: download}:
		}
		return nil
	}
}

func (app *App) contentWriter(ctx context.Context, dc chan Download) func() error {
	return func() error {
		select {
		case <-ctx.Done():
			return nil
		case download := <-dc:
			defer func() { _ = download.Content.Close() }() // Best effort.

			u, err := stdurl.Parse(download.Location)
			if err != nil {
				return errors.Wrap(err, "parsing url")
			}
			if err := os.MkdirAll(path.Dir(u.Path[1:]), os.ModePerm); err != nil {
				return errors.Wrap(err, "making directory")
			}
			f, err := os.Create(u.Path[1:])
			if err != nil {
				return errors.Wrap(err, "creating file")
			}
			defer func() { _ = f.Close() }() // Best effort.

			log.Printf("writing download to %s", u.Path[1:])

			if _, err := io.Copy(f, download.Content); err != nil {
				return errors.Wrap(err, "writing file")
			}
			log.Printf("wrote %s", u.Path[1:])
		}
		return nil
	}
}

func (app *App) list(ctx context.Context) error {
	urls, err := app.urls()
	if err != nil {
		return errors.Wrap(err, "getting urls")
	}
	return json.NewEncoder(os.Stdout).Encode(urls)
}

func (app *App) download(ctx context.Context) error {
	urls, err := app.urls()
	if err != nil {
		return errors.Wrap(err, "getting urls")
	}
	for _, url := range urls {
		// Get the URL's of the actual audio files.
		downloads, err := app.scrape(ctx, url)
		if err != nil {
			return errors.Wrap(err, "scraping audio file URL's")
		}
		// Run the downloads in parallel.
		if err := app.fetch(ctx, downloads); err != nil {
			return errors.Wrap(err, "fetching audio files")
		}
	}
	return nil
}

func (app *App) fetch(ctx context.Context, downloads []string) error {
	var (
		dc      = make(chan Download)
		g, gctx = errgroup.WithContext(ctx)
	)
	for _, dl := range downloads {
		// Spawn goroutines that will fetch each file.
		g.Go(app.contentFetcher(gctx, dl, dc))

		// Spawn goroutines that will write the data to local disk.
		g.Go(app.contentWriter(gctx, dc))
	}
	return g.Wait()
}

func (app *App) scrape(ctx context.Context, url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "fetching "+url)
	}
	defer func() { _ = resp.Body.Close() }() // Best effort.

	root, err := html.Parse(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "parsing html")
	}
	u, err := stdurl.Parse(url)
	if err != nil {
		return nil, errors.Wrap(err, "parsing url")
	}
	var (
		dm    = map[string]struct{}{}
		links = scrape.FindAll(root, scrape.ByTag(atom.A))
	)
	for _, link := range links {
		for _, attr := range link.Attr {
			if attr.Key != "href" || !IsAudioFile(attr.Val) {
				continue
			}
			val := attr.Val

			if strings.HasPrefix(attr.Val, "../") {
				val = attr.Val[3:]
			}
			dm["http://"+u.Host+"/"+val] = struct{}{}
		}
	}
	var downloads []string

	for u := range dm {
		log.Println("going to scrape " + u)
		downloads = append(downloads, u)
	}
	return downloads, nil
}

func (app *App) urls() ([]string, error) {
	var out []string

	if app.Era != "all" {
		sections, ok := app.Samples[app.Era]
		if !ok {
			return nil, errors.New("unsupported era: " + app.Era)
		}
		if len(app.Section) > 0 {
			urls, ok := sections[app.Section]
			if !ok {
				return nil, errors.New("unsupported section: " + app.Section)
			}
			out = append(out, urls...)
		} else {
			for _, urls := range sections {
				out = append(out, urls...)
			}
		}
	} else {
		for _, sections := range app.Samples {
			for _, urls := range sections {
				out = append(out, urls...)
			}
		}
	}
	return out, nil
}

// Config defines the application's configuration.
type Config struct {
	Download bool   `json:"download"`
	Era      string `json:"era"`
	Section  string `json:"section"`

	// Samples is a map from "era" (i.e. pre-2012, post-2012) to "section"
	// (e.g. brass, percussion, woodwind) to the list of URL's that
	// contain the sample download links.
	Samples map[string]map[string][]string `json:"samples"`
}

// NewConfig parses the application's configuration from env/flags.
func NewConfig() (Config, error) {
	config := Config{
		Samples: map[string]map[string][]string{
			"pre-2012": {
				"woodwind": {
					"http://theremin.music.uiowa.edu/MISflute.html",
					"http://theremin.music.uiowa.edu/MISaltoflute.html",
					"http://theremin.music.uiowa.edu/MISbassflute.html",
					"http://theremin.music.uiowa.edu/MISoboe.html",
					"http://theremin.music.uiowa.edu/MISEbclarinet.html",
					"http://theremin.music.uiowa.edu/MISBbclarinet.html",
					"http://theremin.music.uiowa.edu/MISbassclarinet.html",
					"http://theremin.music.uiowa.edu/MISbassoon.html",
					"http://theremin.music.uiowa.edu/MISsopranosaxophone.html",
					"http://theremin.music.uiowa.edu/MISaltosaxophone.html",
				},
				"brass": {
					"http://theremin.music.uiowa.edu/MISFrenchhorn.html",
					"http://theremin.music.uiowa.edu/MISBbtrumpet.html",
					"http://theremin.music.uiowa.edu/MIStenortrombone.html",
					"http://theremin.music.uiowa.edu/MISbasstrombone.html",
					"http://theremin.music.uiowa.edu/MIStuba.html",
				},
				"strings": {
					"http://theremin.music.uiowa.edu/MISviolin.html",
					"http://theremin.music.uiowa.edu/MISviola.html",
					"http://theremin.music.uiowa.edu/MIScello.html",
					"http://theremin.music.uiowa.edu/MISdoublebass.html",
					"http://theremin.music.uiowa.edu/MISviolin2012.html",
					"http://theremin.music.uiowa.edu/MISviola2012.html",
					"http://theremin.music.uiowa.edu/MIScello2012.html",
					"http://theremin.music.uiowa.edu/MISdoublebass2012.html",
				},
				"percussion": {
					"http://theremin.music.uiowa.edu/Mismarimba.html",
					"http://theremin.music.uiowa.edu/MISxylophone.html",
					"http://theremin.music.uiowa.edu/Misvibraphone.html",
					"http://theremin.music.uiowa.edu/MISbells.html",
					"http://theremin.music.uiowa.edu/MIScrotales.html",
					"http://theremin.music.uiowa.edu/MISgongtamtams.html",
					"http://theremin.music.uiowa.edu/MIShandpercussion.html",
					"http://theremin.music.uiowa.edu/MIStambourines.html",
				},
				"piano/other": {
					"http://theremin.music.uiowa.edu/MISpiano.html",
					"http://theremin.music.uiowa.edu/MISballoonpop.html",
					"http://theremin.music.uiowa.edu/MISguitar.html",
				},
				"foundobjects": {
					"http://theremin.music.uiowa.edu/MISfoundobjects1.html",
				},
			},
			"post-2012": {
				"woodwinds": {
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISFlute2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISaltoflute2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISBassFlute2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISOboe2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISEbClarinet2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISBbClarinet2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISBbBassClarinet2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISBassoon2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISBbSopranoSaxophone2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISEbAltoSaxophone2012.html",
				},
				"brass": {
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISHorn2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISBbTrumpet2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISTenorTrombone2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISBassTrombone2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISTuba2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISBbBassClarinet2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISBassoon2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISBbSopranoSaxophone2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISEbAltoSaxophone2012.html",
				},
				"strings": {
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISViolin2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISViola2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISCello2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISDoubleBass2012.html",
				},
				"percussion": {
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISMarimba2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISxylophone2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISVibraphone2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISBells2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISCrotales2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISCymbals2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISGongsTamTams2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISHandPercussion2012.html",
					"http://theremin.music.uiowa.edu/MIS-Pitches-2012/MISTambourines2012.html",
				},
				"foundobjects": {
					"http://theremin.music.uiowa.edu/MISfoundobjects2.html",
				},
			},
		},
	}
	flag.BoolVar(&config.Download, "dl", false, "Download samples (default is to just print a JSON list to stdout).")
	flag.StringVar(&config.Era, "e", "all", "Filter by era ('all', 'pre-2012', 'post-2012').")
	flag.StringVar(&config.Section, "s", "", "(REQUIRED) Section (e.g. brass, woodwind, percussion")
	flag.Parse()

	if config.Era != "all" {
		sections, ok := config.Samples[config.Era]
		if !ok {
			return config, errors.New("unsupported era: " + config.Era)
		}
		// Validate -s if it was provided.
		if len(config.Section) > 0 {
			if _, ok := sections[config.Section]; !ok {
				return config, errors.New("unsupported section: " + config.Section)
			}
		}
	}
	return config, nil
}

// Download represents a single audio file download.
type Download struct {
	Content  io.ReadCloser
	Location string
}

// IsAudioFile returns true if the provided string ends with .aif or .aiff or .wav
func IsAudioFile(s string) bool {
	if strings.HasSuffix(s, ".aif") || strings.HasSuffix(s, ".aiff") || strings.HasSuffix(s, ".wav") {
		return true
	}
	return false
}
