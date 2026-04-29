package main

import (
	"fmt"
	"os"

	"github.com/oauth2-proxy/oauth2-proxy/v7/pkg/apis/options"
	"github.com/oauth2-proxy/oauth2-proxy/v7/pkg/validation"
	"github.com/spf13/pflag"
)

func main() {
	log := logger.NewLogEntry()

	flagSet := pflag.NewFlagSet("oauth2-proxy", pflag.ExitOnError)

	// Define core flags
	config := flagSet.String("config", "", "path to config file")
	showVersion := flagSet.Bool("version", false, "print version string")
	convertConfig := flagSet.Bool("convert-config-to-alpha", false,
		"if true, the proxy will load the configuration as normal and convert the config to the new alpha format, then exit")

	options.RegisterLegacyFlagSet(flagSet)

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		log.WithError(err).Fatal("failed to parse flags")
	}

	if *showVersion {
		fmt.Printf("oauth2-proxy %s (built with %s)\n", VERSION, runtime.Version())
		return
	}

	opts, err := options.Load(*config, flagSet)
	if err != nil {
		log.WithError(err).Fatal("failed to load configuration")
	}

	if *convertConfig {
		if err := printConvertedConfig(opts); err != nil {
			log.WithError(err).Fatal("failed to convert configuration")
		}
		return
	}

	if err := validation.Validate(opts); err != nil {
		log.WithError(err).Fatal("invalid configuration")
	}

	oauthProxy, err := proxy.NewOAuthProxy(opts, func(email string) bool {
		return opts.IsValidatedEmail(email)
	})
	if err != nil {
		log.WithError(err).Fatal("failed to initialize oauth2 proxy")
	}

	server := server.NewServer(opts, oauthProxy)
	if err := server.Start(signalCtx()); err != nil {
		log.WithError(err).Fatal("server exited with error")
	}
}

// signalCtx returns a context that is cancelled on SIGINT or SIGTERM.
func signalCtx() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx
}

// printConvertedConfig marshals the loaded options into the alpha config
// format and writes it to stdout.
func printConvertedConfig(opts *options.Options) error {
	alpha, err := options.ConvertToAlpha(opts)
	if err != nil {
		return fmt.Errorf("converting config: %w", err)
	}
	out, err := yaml.Marshal(alpha)
	if err != nil {
		return fmt.Errorf("marshalling alpha config: %w", err)
	}
	_, err = os.Stdout.Write(out)
	return err
}
