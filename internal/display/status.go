package display

import (
	"fmt"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

func Status(msg string) {
	fmt.Printf("%s%s▸%s %s\n", colorBold, colorCyan, colorReset, msg)
}

func Success(msg string) {
	fmt.Printf("%s%s✓%s %s\n", colorBold, colorGreen, colorReset, msg)
}

func Warn(msg string) {
	fmt.Printf("%s%s!%s %s\n", colorBold, colorYellow, colorReset, msg)
}

func Error(msg string) {
	fmt.Printf("%s%s✗%s %s\n", colorBold, colorRed, colorReset, msg)
}

func Info(label, value string) {
	fmt.Printf("  %s%-18s%s %s\n", colorCyan, label, colorReset, value)
}

// Countdown displays a live in-place countdown timer.
// It returns when the deadline is reached or the done channel is closed.
func Countdown(deadline time.Time, done <-chan struct{}) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		remaining := time.Until(deadline).Truncate(time.Second)
		if remaining <= 0 {
			fmt.Printf("\r%s%s⏱  TTL expired%s                    \n", colorBold, colorYellow, colorReset)
			return
		}

		fmt.Printf("\r%s%s⏱  %s remaining%s     ", colorBold, colorCyan, remaining, colorReset)

		select {
		case <-done:
			fmt.Println()
			return
		case <-ticker.C:
		}
	}
}
