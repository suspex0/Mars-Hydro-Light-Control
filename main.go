package main

import (
	"lightcontrol/internal/api"
	"lightcontrol/internal/config"
	"log"
	"time"
)

func main() {
	// Load configurations via new config package.
	lt, err := config.LoadLightTimerConfig("timer.json")
	if err != nil {
		log.Fatal(err)
	}
	mhAccount, err := config.LoadMHAccountConfig("account.json")
	if err != nil {
		log.Fatal(err)
	}

	// Print computed timing data for visualization.
	lt.PrintTimingData(time.Now())

	// Assume initial state unknown; force update on startup.
	lastBrightness := -1

	// Function to update lamp state if needed.
	updateState := func(state int) {
		// Always update if lastBrightness is unknown (-1)
		if lastBrightness != -1 && state == lastBrightness {
			if state == 0 {
				log.Println("Lamp remains OFF")
			} else {
				log.Printf("Lamp remains ON at %d%% brightness", state)
			}
			return
		}
		log.Printf("Setting lamp to %d%% brightness", state)
		mhapi := api.NewMarsHydroAPI(mhAccount.Email, mhAccount.Password, mhAccount.Wifiname, mhAccount.Timezone, mhAccount.Language)
		if err := mhapi.Login(); err != nil {
			log.Println("Login error:", err)
			return
		}
		if err := mhapi.SetBrightness(state); err != nil {
			log.Println("Failed to set brightness:", err)
			return
		}
		lastBrightness = state
		if state == 0 {
			log.Println("Status: Lamp OFF")
		} else {
			log.Printf("Status: Lamp ON at %d%% brightness", state)
		}
	}

	// On startup: set the lamp to expected state.
	now := time.Now()
	expected := lt.GetExpectedBrightness(now)
	updateState(expected)

	// Start a loop that every minute checks and updates the lamp state if needed.
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	log.Println("Configuration loaded, starting event loop...")
	for now := range ticker.C {
		expected := lt.GetExpectedBrightness(now)
		updateState(expected)
	}
}
