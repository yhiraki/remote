package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/yhiraki/remote/internal/command"
	"github.com/yhiraki/remote/internal/config"
	"github.com/yhiraki/remote/internal/host"
)

var (
	home string
	cwd  string
)

func init() {
	var err error

	home, err = os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	cwd, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
}

type stringSlice []string

func (i *stringSlice) String() string {
	return fmt.Sprintf("%v", *i)
}

func (i *stringSlice) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func _main() error {
	cfg, err := config.New()
	if err != nil {
		return err
	}

	configName := ".remoterc.json"
	if err := cfg.Load(configName); err != nil {
		log.Printf("%s could not parsed.", configName)
		return err
	}

	// create directories
	for _, d := range []string{cfg.CacheDir, cfg.ConfigDir} {
		if _, err := os.Stat(d); err != nil {
			if err = os.MkdirAll(d, 0o705); err != nil {
				return err
			}
		}
	}

	// command line parsing
	var envVars stringSlice
	flag.Var(&envVars, "e", "set environment variable (e.g. -e KEY=VALUE)")
	flag.Var(&envVars, "env", "set environment variable (e.g. --env KEY=VALUE)")
	isDryRun := flag.Bool("dry-run", false, "dry run")
	isVerbose := flag.Bool("verbose", false, "enable verbose logging")
	showVersion := flag.Bool("version", false, "print version information")
	isBackground := flag.Bool("background", false, "run tunnel in background")
	flag.Parse()

	if *showVersion {
		if info, ok := debug.ReadBuildInfo(); ok {
			fmt.Println(info.Main.Version)
		} else {
			fmt.Println("version not found")
		}
		return nil
	}

	// get hostname
	h := cfg.Hostname
	if cfg.HostnameCommand != "" {
		h, err = host.Get(
			cfg.HostnameCommand,
			filepath.Join(cfg.CacheDir, "hostname"),
			cfg.CacheExpireMinutes,
			*isVerbose)
		if err != nil {
			return err
		}
	}

	// get relative current path
	cwdRel, err := filepath.Rel(home, cwd)
	if err != nil {
		return err
	}

	return command.Run(cfg, h, flag.Args(), envVars, *isDryRun, *isBackground, *isVerbose, cwdRel)
}

func main() {
	if err := _main(); err != nil {
		fmt.Println(err)
	}
}
