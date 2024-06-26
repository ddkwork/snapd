// -*- Mode: Go; indent-tabs-mode: t -*-
//go:build !nomanagers

/*
 * Copyright (C) 2020 Canonical Ltd
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
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ddkwork/golibrary/mylog"
	"github.com/snapcore/snapd/dirs"
)

func handleCertConfiguration(tr RunTransaction, opts *fsOnlyContext) error {
	// This handles the "snap revert core" case:
	// We need to go over each pem cert on disk and check if there is
	// a matching config entry - if not->delete the cert
	//
	// XXX: remove this code once we have a general way to handle
	//      "snap revert" and config updates
	//
	// TODO: add ways to detect cleanly if tr is a patch, skip the sync code if it is
	storeCerts := mylog.Check2(filepath.Glob(filepath.Join(dirs.SnapdStoreSSLCertsDir, "*.pem")))

	for _, storeCertPath := range storeCerts {
		optionName := strings.TrimSuffix(filepath.Base(storeCertPath), ".pem")
		v := mylog.Check2(coreCfg(tr, "store-certs."+optionName))

		if v == "" {
			mylog.Check(os.Remove(storeCertPath))
		}
	}

	// add/remove regular (non revert) changes
	for _, name := range tr.Changes() {
		if !strings.HasPrefix(name, "core.store-certs.") {
			continue
		}

		nameWithoutSnap := strings.SplitN(name, ".", 2)[1]
		cert := mylog.Check2(coreCfg(tr, nameWithoutSnap))

		optionName := strings.SplitN(name, ".", 3)[2]
		certPath := filepath.Join(dirs.SnapdStoreSSLCertsDir, optionName+".pem")
		switch cert {
		case "":
			// remove
			if mylog.Check(os.Remove(certPath)); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("cannot remove store certificate: %v", err)
			}
		default:
			mylog.Check(os.MkdirAll(dirs.SnapdStoreSSLCertsDir, 0755))
			mylog.Check(os.WriteFile(certPath, []byte(cert), 0644))

		}
	}

	return nil
}

func validateCertSettings(tr RunTransaction) error {
	for _, name := range tr.Changes() {
		if !strings.HasPrefix(name, "core.store-certs.") {
			continue
		}

		nameWithoutSnap := strings.SplitN(name, ".", 2)[1]
		cert := mylog.Check2(coreCfg(tr, nameWithoutSnap))

		if cert != "" {
			optionName := strings.SplitN(name, ".", 3)[2]
			if !validCertName(optionName) {
				return fmt.Errorf("cannot set store ssl certificate under name %q: name must only contain word characters or a dash", optionName)
			}
			cp := x509.NewCertPool()
			if !cp.AppendCertsFromPEM([]byte(cert)) {
				return fmt.Errorf("cannot decode pem certificate %q", optionName)
			}
		}
	}

	return nil
}
