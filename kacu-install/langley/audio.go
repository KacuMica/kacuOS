// audio.go — Step 5: audio stack choice (archinstall's "Audio" step).
package main

import (
	"strings"
)

// audioOption is one selectable audio server configuration.
type audioOption struct {
	id     string
	label  string
	add    []string // packages to add
	remove []string // package-name prefixes to drop from the base list
}

var audioOptions = []audioOption{
	{"pipewire", "PipeWire (recommended)", []string{"pipewire", "pipewire-pulse", "pipewire-alsa", "wireplumber"}, []string{"pulseaudio"}},
	{"pulse", "PulseAudio", []string{"pulseaudio", "pulseaudio-alsa", "pulseaudio-bluetooth"}, []string{"pipewire", "wireplumber"}},
	{"alsa", "ALSA only (no audio daemon)", nil, []string{"pipewire", "wireplumber", "pulseaudio"}},
}

// audioStep lists detected playback hardware and picks the audio stack that
// the installed system will run; the output device itself is managed by the
// desktop (GNOME sound panel) after boot.
func audioStep(cfg *Config) {
	if out, err := runCapture([]string{"aplay", "-l"}, ""); err == nil {
		var cards []string
		for _, line := range strings.Split(out, "\n") {
			if strings.Contains(line, "card") {
				cards = append(cards, strings.TrimSpace(line))
			}
		}
		if len(cards) > 0 {
			plog("Detected sound cards:")
			for _, card := range cards {
				plog("  %s", card)
			}
		} else {
			plog("No sound cards detected (drivers can be added after install)")
		}
	} else {
		plog("aplay is unavailable; skipping sound card detection")
	}

	labels := make([]string, len(audioOptions))
	for i, option := range audioOptions {
		labels[i] = option.label
	}
	choice := menu("Audio server (installed system)", labels, false)
	option := audioOptions[choice-1]
	cfg.AudioID = option.id
}

// audioOptionByID returns the option for id, defaulting to pipewire.
func audioOptionByID(id string) audioOption {
	for _, option := range audioOptions {
		if option.id == id {
			return option
		}
	}
	return audioOptions[0]
}

// audioLabel returns a display name for the plan output.
func audioLabel(id string) string {
	return audioOptionByID(id).label
}
