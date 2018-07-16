package selfupdate

import (
	"fmt"
	"os"
	"time"

	"../utils"
	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

func StartSelfupdate(slug string, version string) {
	selfUpdateTicker := time.NewTicker(5 * time.Minute)
	go func(selfUpdateTicker *time.Ticker) {
		for {
			select {
			case <-selfUpdateTicker.C:
				if err := selfUpdate(slug, version); err != nil {
					fmt.Fprintln(os.Stderr, err)
					// os.Exit(1)
				}
			}
		}
	}(selfUpdateTicker)
}

func selfUpdate(slug string, version string) error {
	previous := semver.MustParse(version)
	latest, err := selfupdate.UpdateSelf(previous, slug)
	if err != nil {
		return err
	}

	if !previous.Equals(latest.Version) {
		fmt.Println("Update successfully done to version", latest.Version)
		// fmt.Println("Release note:\n", latest.ReleaseNotes)

		utils.Restart()
	}

	return nil
}
