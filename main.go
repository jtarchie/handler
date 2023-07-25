package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type CLI struct {
	Port int `help:"port for the server" default:"8080"`
}

func (c *CLI) Run() error {
	e := echo.New()
	e.Use(middleware.Logger())

	ctx := context.Background()

	// Prepare a cache directory.
	cacheDir, err := os.MkdirTemp("", "example")
	if err != nil {
		return fmt.Errorf("could not create cache dir: %w", err)
	}
	defer os.RemoveAll(cacheDir)

	// Initializes the new compilation cache with the cache directory.
	// This allows the compilation caches to be shared even across multiple OS processes.
	cache, err := wazero.NewCompilationCacheWithDir(cacheDir)
	if err != nil {
		return fmt.Errorf("could not create cache: %w", err)
	}
	defer cache.Close(ctx)

	// Creates a shared runtime config to share the cache across multiple wazero.Runtime.
	runtimeConfig := wazero.
		NewRuntimeConfig().
		WithCompilationCache(cache).
		WithCloseOnContextDone(true)

	// Creates wazero.Runtime  with the same compilation cache.
	runtime := wazero.NewRuntimeWithConfig(ctx, runtimeConfig)
	defer runtime.Close(ctx)

	wasi_snapshot_preview1.MustInstantiate(ctx, runtime)

	matches, err := doublestar.FilepathGlob("examples/**/main.wasm")
	if err != nil {
		return fmt.Errorf("could not glob examples: %w", err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("could not find any wasm modules")
	}

	compiledContents := map[string][]byte{}
	for _, match := range matches {
		contents, err := os.ReadFile(match)
		if err != nil {
			return fmt.Errorf("could nto read file (%q): %w", match, err)
		}

		parts := strings.Split(filepath.Dir(match), "/")
		name := parts[len(parts)-1]
		compiledContents[name] = contents
		fmt.Printf("parts = %#v\n", parts)
	}

	e.GET("/:module_name", func(c echo.Context) error {
		lookup := c.Param("module_name")
		compiledWasm, ok := compiledContents[lookup]

		if !ok {
			return fmt.Errorf("could not find module (%q)", lookup)
		}

		config := wazero.
			NewModuleConfig().
			WithStdin(c.Request().Body).
			WithStdout(c.Response().Unwrap()).
			WithStderr(os.Stderr).
			WithEnv("REQUEST_METHOD", "GET").
			WithEnv("SERVER_PROTOCOL", "HTTP/1.1").
			WithName("")

		ctx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
		defer cancel()

		_, err := runtime.InstantiateWithConfig(ctx, compiledWasm, config)
		if err != nil {
			return fmt.Errorf("could not init module (%q): %w", lookup, err)
		}

		return nil
	})

	return e.Start(fmt.Sprintf(":%d", c.Port))
}

func main() {
	cli := &CLI{}
	ctx := kong.Parse(cli,
		kong.UsageOnError(),
	)
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}
