// software.go — Step 6: browser, Steam, extra apps and kacu-pkg boxes.
package main

var extraApps = []string{
	"libreoffice-fresh", // office suite
	"thunderbird",       // mail client
	"vlc",               // media player
	"gimp",              // image editor
	"obs-studio",        // recording / streaming
	"blender",           // 3D modeling
	"code",              // Visual Studio Code OSS build (extra repo)
	"kdenlive",          // video editor
}

// softwareStep covers everything that goes beyond the base system.
func softwareStep(cfg *Config) {
	browser := menu("Browser", []string{
		"Firefox (official repos)",
		"Chromium (official repos)",
		"Brave (AUR: brave-bin, built at the end of the install)",
		"None",
	}, false)
	switch browser {
	case 1:
		cfg.BrowserRepo = "firefox"
	case 2:
		cfg.BrowserRepo = "chromium"
	case 3:
		cfg.BrowserAUR = "brave-bin"
		warnf("Brave comes from the AUR and needs an AUR helper; " +
			"langley installs paru automatically (falls back to a warning).")
	}

	if confirmYes("Install Steam? (this enables the multilib repository)") {
		cfg.Steam = true
	}

	for _, idx := range multiMenu("Extra software", extraApps, nil) {
		cfg.Apps = append(cfg.Apps, extraApps[idx])
	}

	for _, idx := range multiMenu("kacu-pkg boxes (optional presets)", availableBoxes, nil) {
		cfg.Boxes = append(cfg.Boxes, availableBoxes[idx])
	}
}
