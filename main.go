package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/pkg/errors"
)

func main() {
	config, err := NewConfig()
	die(errors.Wrap(err, "parsing config"))
	app, err := NewApp(config)
	die(errors.Wrap(err, "initializing app"))
	die(app.Run(context.Background()))
}

// App defines the application's behavior.
type App struct {
	Config
}

// NewApp initializes the application.
func NewApp(conf Config) (*App, error) {
	app := &App{Config: conf}

	return app, nil
}

// Run runs the application.
func (a *App) Run(ctx context.Context) error {
	if a.Download {
		return a.download(ctx)
	}
	return a.list(ctx)
}

func (a *App) download(ctx context.Context) error {
	return nil
}

func (a *App) list(ctx context.Context) error {
	urls, err := a.urls()
	if err != nil {
		return errors.Wrap(err, "getting urls")
	}
	return json.NewEncoder(os.Stdout).Encode(urls)
}

func (a *App) urls() ([]string, error) {
	var out []string

	if a.Era != "all" {
		sections, ok := a.Samples[a.Era]
		if !ok {
			return nil, errors.New("unsupported era: " + a.Era)
		}
		if len(a.Section) > 0 {
			urls, ok := sections[a.Section]
			if !ok {
				return nil, errors.New("unsupported section: " + a.Section)
			}
			out = append(out, urls...)
		} else {
			for _, urls := range sections {
				out = append(out, urls...)
			}
		}
	} else {
		for _, sections := range a.Samples {
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

func die(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
