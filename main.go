package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/urfave/cli"
)

type Colors struct {
	Purple     int64
	DarkPurple int64
	Green      int64
	Red        int64
	Yellow     int64
}

var opts struct {
	Debug        bool
	Interval     time.Duration
	Repositories cli.StringSlice
	BitBucket    struct {
		Username string
		Password string
	}
	Colors Colors
	Hue    struct {
		ApiKey string
	}
}

func main() {
	app := cli.NewApp()
	app.Action = run
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "a,api-key",
			EnvVar:      "API_KEY",
			Usage:       "api key for phillips hue bridge",
			Destination: &opts.Hue.ApiKey,
		},
		cli.BoolFlag{
			Name:        "d,debug",
			EnvVar:      "DEBUG",
			Usage:       "print additional debug messages",
			Destination: &opts.Debug,
		},
		cli.StringFlag{
			Name:        "u,username",
			Usage:       "bitbucket username",
			EnvVar:      "BITBUCKET_USERNAME",
			Destination: &opts.BitBucket.Username,
		},
		cli.StringFlag{
			Name:        "p,password",
			Usage:       "bitbucket password",
			EnvVar:      "BITBUCKET_PASSWORD",
			Destination: &opts.BitBucket.Password,
		},
		cli.Int64Flag{
			Name:        "green",
			Value:       28000,
			Usage:       "set saturation of green",
			EnvVar:      "GREEN",
			Destination: &opts.Colors.Green,
		},
		cli.Int64Flag{
			Name:        "purple",
			Value:       43000,
			Usage:       "set saturation of purple",
			EnvVar:      "BLUE",
			Destination: &opts.Colors.Purple,
		},
		cli.Int64Flag{
			Name:        "dark-purple",
			Value:       46920,
			Usage:       "set saturation of dark purple",
			EnvVar:      "BLUE",
			Destination: &opts.Colors.DarkPurple,
		},
		cli.Int64Flag{
			Name:        "red",
			Value:       0,
			Usage:       "set saturation of red",
			EnvVar:      "RED",
			Destination: &opts.Colors.Red,
		},
		cli.Int64Flag{
			Name:        "yellow",
			Value:       15000,
			Usage:       "set saturation of yellow",
			EnvVar:      "YELLOW",
			Destination: &opts.Colors.Yellow,
		},
		cli.DurationFlag{
			Name:        "interval",
			Value:       time.Minute,
			Usage:       "interval between polling bitbucket",
			EnvVar:      "INTERVAL",
			Destination: &opts.Interval,
		},
		cli.StringSliceFlag{
			Name:   "r,repo",
			Usage:  "bitbucket repositories to include owner/slug e.g. realogy_corp/role-user-management",
			EnvVar: "REPOSITORIES",
			Value:  &opts.Repositories,
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatalln(err)
	}
}

func run(_ *cli.Context) error {
	addr, err := discover()
	if err != nil {
		return err
	}

	fn := manageColor(opts.Hue.ApiKey, addr, opts.Colors)

	fmt.Println(err)
	for _, repo := range opts.Repositories {
		go pollBuildStatus(opts.BitBucket.Username, opts.BitBucket.Password, repo, opts.Interval, fn)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Kill, os.Interrupt)

	<-stop

	return nil
}
