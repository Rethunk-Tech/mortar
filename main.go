package main

import (
	"embed"
	"log"
	"os"

	"github.com/Rethunk-AI/mortar/internal/games"
	"github.com/Rethunk-AI/mortar/internal/profile"
	"github.com/Rethunk-AI/mortar/internal/settings"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	store, err := settings.Open()
	if err != nil {
		log.Fatal(err)
	}
	svc := settings.NewService(store)

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	gamesSvc := games.NewService(home)

	profiles, err := profile.Open()
	if err != nil {
		log.Fatal(err)
	}

	var window *application.WebviewWindow

	app := application.New(application.Options{
		Name:        "Mortar",
		Description: "Multi-game desktop mod manager",
		Services: []application.Service{
			application.NewService(svc), application.NewService(gamesSvc),
			application.NewService(profile.NewService(profiles)),
		},
		Assets: application.AssetOptions{
			Handler:    application.AssetFileServerFS(assets),
			Middleware: games.ArtMiddleware(home),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "tech.rethunk.mortar",
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				window.Restore()
				window.Focus()
			},
		},
	})

	svc.App = app

	// Wails fixes BackgroundType at window creation, so the stored value applies on restart.
	background, colour := application.BackgroundTypeSolid, application.NewRGBA(25, 25, 30, 255)
	if store.Get().Translucent {
		background, colour = application.BackgroundTypeTranslucent, application.NewRGBA(25, 25, 30, 204)
	}

	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Mortar",
		Width:            1280,
		Height:           720,
		MinWidth:         768,
		MinHeight:        432,
		Frameless:        true,
		BackgroundType:   background,
		BackgroundColour: colour,
		Windows: application.WindowsWindow{
			BackdropType: application.Acrylic,
		},
		URL: "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
