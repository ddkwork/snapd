// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2022 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package configcore

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ddkwork/golibrary/mylog"
	"github.com/snapcore/snapd/dirs"
	"github.com/snapcore/snapd/osutil"
	"github.com/snapcore/snapd/overlord/configstate/config"
	"github.com/snapcore/snapd/sandbox/apparmor"
	"github.com/snapcore/snapd/snap/sysparams"
	"github.com/snapcore/snapd/sysconfig"
)

// For mocking in tests
var (
	osutilEnsureFileState = osutil.EnsureFileState
	osutilDirExists       = osutil.DirExists

	apparmorUpdateHomedirsTunable    = apparmor.UpdateHomedirsTunable
	apparmorReloadAllSnapProfiles    = apparmor.ReloadAllSnapProfiles
	apparmorSetupSnapConfineSnippets = apparmor.SetupSnapConfineSnippets
)

// No path located under any of these top-level directories can be used.
// This is not a security measure (the configuration can only be changed by
// the system administrator anyways) but rather a check around unintended
// mistakes.
var invalidPrefixes = []string{
	"/bin/",
	"/boot/",
	"/dev/",
	"/etc/",
	"/lib/",
	"/proc/",
	"/root/",
	"/sbin/",
	"/sys/",
	"/tmp/",
}

func init() {
	// add supported configuration of this module
	supportedConfigurations["core.homedirs"] = true
}

func configureHomedirsInAppArmorAndReload(homedirs []string, opts *fsOnlyContext) error {
	// important to note here that when a configure hook is invoked, handlers are invoked,
	// so they are not *just* invoked by user-interaction, and we do not want to break those
	// actions. So no-op on systems that do not support apparmor.
	if apparmor.ProbedLevel() != apparmor.Full && apparmor.ProbedLevel() != apparmor.Partial {
		return nil
	}
	mylog.Check(apparmorUpdateHomedirsTunable(homedirs))

	// Only update snap-confine apparmor snippets and reload profiles
	// if it's during runtime
	if opts == nil {
		mylog.Check2(apparmorSetupSnapConfineSnippets())
		mylog.Check(

			// We must reload the apparmor profiles in order for our changes to become
			// effective. In theory, all profiles are affected; in practice, we are a
			// bit egoist and only care about snapd.
			apparmorReloadAllSnapProfiles())

	}
	return nil
}

func updateHomedirsConfig(config string, opts *fsOnlyContext) error {
	// if opts is not nil this is image build time
	rootDir := dirs.GlobalRootDir
	if opts != nil {
		rootDir = opts.RootDir
	}
	sspPath := dirs.SnapSystemParamsUnder(rootDir)
	mylog.Check(os.MkdirAll(path.Dir(sspPath), 0755))

	ssp := mylog.Check2(sysparams.Open(rootDir))

	ssp.Homedirs = config
	mylog.Check(ssp.Write())

	// Update value in dirs
	homedirs := dirs.SetSnapHomeDirs(config)
	return configureHomedirsInAppArmorAndReload(homedirs, opts)
}

func handleHomedirsConfiguration(dev sysconfig.Device, tr ConfGetter, opts *fsOnlyContext) error {
	conf := mylog.Check2(coreCfg(tr, "homedirs"))

	var prevConfig string
	if mylog.Check(tr.GetPristine("core", "homedirs", &prevConfig)); err != nil && !config.IsNoOption(err) {
		return err
	}
	if conf == prevConfig {
		return nil
	}

	// XXX: Check after verifying no change is done to the actual option as the handler
	// is still invoked.
	if !dev.Classic() {
		// There is no specific reason this can not be supported on core, but to
		// remove this check, we need a spread test verifying this does indeed work
		// on core as well.
		return fmt.Errorf("configuration of homedir locations on Ubuntu Core is currently unsupported. Please report a bug if you need it")
	}
	mylog.Check(updateHomedirsConfig(conf, opts))

	return nil
}

func validateHomedirsConfiguration(tr ConfGetter) error {
	config := mylog.Check2(coreCfg(tr, "homedirs"))

	if config == "" {
		return nil
	}

	homedirs := strings.Split(config, ",")
	for _, dir := range homedirs {
		if !filepath.IsAbs(dir) {
			return fmt.Errorf("path %q is not absolute", dir)
		}
		mylog.Check(

			// Ensure that the paths are not too fancy, as this could cause
			// AppArmor to interpret them as patterns
			apparmor.ValidateNoAppArmorRegexp(dir))

		// Also make sure that the path is not going to interfere with standard
		// locations
		if dir[len(dir)-1] != '/' {
			dir += "/"
		}
		for _, prefix := range invalidPrefixes {
			if strings.HasPrefix(dir, prefix) {
				return fmt.Errorf("path %q uses reserved root directory %q", dir, prefix)
			}
		}

		exists, isDir := mylog.Check3(osutilDirExists(dir))

		if !exists {
			// There's no harm in letting this pass even if the directory does
			// not exist, as long as snap-confine handles it properly. But for
			// the time being let's err on the safe side.
			return fmt.Errorf("path %q does not exist", dir)
		}
		if !isDir {
			return fmt.Errorf("path %q is not a directory", dir)
		}
	}

	return err
}
