package timer

import (
	"encoding/json"
	"errors"
	"log"
	"math"
	"os"
	"time"
)

type LightTimer struct {
	StartHour     int `json:"StartHour"`
	PlateauHour   int `json:"PlateauHour"`
	EndHour       int `json:"EndHour"`
	StepSize      int `json:"StepSize"`
	PlateauOffset int `json:"PlateauOffset"` // offset in hours applied to sunrise/sunset ramp durations
	Brightness    int `json:"Brightness"`    // new: maximum brightness at plateau phase
}

func (lt *LightTimer) SaveConfig(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	return encoder.Encode(lt)
}

func (lt *LightTimer) LoadConfig(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(lt); err != nil {
		return err
	}
	// Basic validation: EndHour must be greater than StartHour + PlateauHour
	if lt.EndHour <= lt.StartHour+lt.PlateauHour {
		return errors.New("invalid configuration: EndHour must be greater than StartHour + PlateauHour")
	}
	// New: Validate that StepSize is a multiple of 5.
	if lt.StepSize%5 != 0 {
		return errors.New("invalid configuration: StepSize must be a multiple of 5")
	}
	return nil
}

// GetExpectedBrightness calculates the desired brightness based on current time and configuration.
// Assumptions:
// - Before StartHour or after EndHour → 0
// - Sunrise phase: from StartHour to (StartHour + sunriseRamp) with brightness rising linearly from StepSize to lt.Brightness.
// - Plateau phase: brightness equals lt.Brightness.
// - Sunset phase: linear decrease from lt.Brightness to StepSize before turning off.
// The sunrise and sunset durations are adjusted by PlateauOffset (in hours). For example, a negative PlateauOffset
// shortens sunrise and lengthens sunset, keeping plateau duration constant.
func (lt *LightTimer) GetExpectedBrightness(now time.Time) int {
	nowSec := now.Hour()*3600 + now.Minute()*60 + now.Second()
	startSec := lt.StartHour * 3600
	endSec := lt.EndHour * 3600
	plateauSec := lt.PlateauHour * 3600

	// Baseline ramp duration (in seconds) for sunrise and sunset without offset.
	baselineRamp := float64(endSec-startSec-plateauSec) / 2.0
	// Convert PlateauOffset (in hours) to seconds.
	offsetSec := float64(lt.PlateauOffset * 3600)
	// Adjust sunrise and sunset durations.
	sunriseRamp := baselineRamp + offsetSec // if PlateauOffset is negative, sunriseRamp is shorter.
	sunsetRamp := baselineRamp - offsetSec  // and sunsetRamp becomes longer.

	if nowSec < startSec || nowSec >= endSec {
		return 0
	}
	// Sunrise phase: from startSec to (startSec + sunriseRamp)
	if float64(nowSec) < float64(startSec)+sunriseRamp {
		fraction := float64(nowSec-startSec) / sunriseRamp
		brightness := float64(lt.StepSize) + fraction*(float64(lt.Brightness)-float64(lt.StepSize))
		result := int(math.Round(brightness/float64(lt.StepSize)) * float64(lt.StepSize))
		if result > lt.Brightness {
			result = lt.Brightness
		}
		return result
	}

	// Plateau phase: constant brightness equals lt.Brightness.
	plateauStart := startSec + int(sunriseRamp)
	plateauEnd := plateauStart + plateauSec
	if nowSec >= plateauStart && nowSec < plateauEnd {
		return lt.Brightness
	}

	// Sunset phase: from plateauEnd to endSec.
	if nowSec < endSec {
		fraction := float64(endSec-nowSec) / sunsetRamp
		brightness := float64(lt.StepSize) + fraction*(float64(lt.Brightness)-float64(lt.StepSize))
		result := int(math.Round(brightness/float64(lt.StepSize)) * float64(lt.StepSize))
		if result > lt.Brightness {
			result = lt.Brightness
		}
		return result
	}
	return 0
}

// PrintTimingData prints computed timing information using current config and reference time.
func (lt *LightTimer) PrintTimingData(now time.Time) {
	startSec := lt.StartHour * 3600
	endSec := lt.EndHour * 3600
	plateauSec := lt.PlateauHour * 3600

	// Baseline ramp duration (in seconds) for sunrise and sunset without offset.
	baselineRamp := float64(endSec-startSec-plateauSec) / 2.0
	// Convert PlateauOffset (in hours) to seconds.
	offsetSec := float64(lt.PlateauOffset * 3600)
	// Adjust sunrise and sunset durations.
	sunriseRamp := baselineRamp + offsetSec
	sunsetRamp := baselineRamp - offsetSec

	plateauStart := startSec + int(sunriseRamp)
	plateauEnd := plateauStart + plateauSec

	log.Printf("Timing Data for reference time %v:", now.Format("15:04:05"))
	log.Printf("  Baseline ramp (seconds): %.0f", baselineRamp)
	log.Printf("  PlateauOffset (seconds): %.0f", offsetSec)
	log.Printf("  Sunrise ramp duration (sec): %.0f", sunriseRamp)
	log.Printf("  Sunset ramp duration (sec): %.0f", sunsetRamp)
	log.Printf("  Sunrise at: %02d:%02d:%02d", startSec/3600, (startSec/60)%60, startSec%60)
	log.Printf("  Plateau starts at: %02d:%02d:%02d", plateauStart/3600, (plateauStart/60)%60, plateauStart%60)
	log.Printf("  Plateau brightness: %d%%", lt.Brightness)
	log.Printf("  Plateau ends at: %02d:%02d:%02d", plateauEnd/3600, (plateauEnd/60)%60, plateauEnd%60)
	log.Printf("  Sunset at: %02d:%02d:%02d", endSec/3600, (endSec/60)%60, endSec%60)
}
