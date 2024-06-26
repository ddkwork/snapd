// -*- Mode: Go; indent-tabs-mode: t -*-

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

package randutil

import (
	cryptorand "crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/ddkwork/golibrary/mylog"
)

// CryptoTokenBytes returns a crypthographically secure token of
// nbytes random bytes.
func CryptoTokenBytes(nbytes int) ([]byte, error) {
	b := make([]byte, nbytes)
	_ := mylog.Check2(cryptorand.Read(b))

	return b, nil
}

// CryptoToken returns a crypthographically secure token string
// encoding nbytes random bytes.
// The result is URL-safe.
func CryptoToken(nbytes int) (string, error) {
	b := mylog.Check2(CryptoTokenBytes(nbytes))

	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Allow mocking of the path through an exported reference.
var kernelUUIDPath = "/proc/sys/kernel/random/uuid"

// RandomKernelUUID will return a UUID from the kernel's procfs API at
// /proc/sys/kernel/random/uuid. Only to be used in very specific uses, most
// random code should use CryptoToken(Bytes) instead.
func RandomKernelUUID() (string, error) {
	b := mylog.Check2(os.ReadFile(kernelUUIDPath))

	return strings.TrimSpace(string(b)), nil
}
