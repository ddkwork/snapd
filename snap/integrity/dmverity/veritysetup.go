// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2023 Canonical Ltd
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

package dmverity

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/ddkwork/golibrary/mylog"
	"github.com/snapcore/snapd/logger"
	"github.com/snapcore/snapd/osutil"
)

// Info represents the dm-verity related data that:
//  1. are not included in the superblock which is generated by default when running
//     veritysetup.
//  2. need their authenticity verified prior to loading the integrity data into the
//     kernel.
//
// For now, since we are keeping the superblock as it is, this only includes the root hash.
type Info struct {
	RootHash string `json:"root-hash"`
}

func getVal(line string) (string, error) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("internal error: unexpected veritysetup output format")
	}
	return strings.TrimSpace(parts[1]), nil
}

func getRootHashFromOutput(output []byte) (rootHash string, err error) {
	scanner := bufio.NewScanner(bytes.NewBuffer(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Root hash") {
			rootHash = mylog.Check2(getVal(line))
		}
		if strings.HasPrefix(line, "Hash algorithm") {
			hashAlgo := mylog.Check2(getVal(line))

			if hashAlgo != "sha256" {
				return "", fmt.Errorf("internal error: unexpected hash algorithm")
			}
		}
	}
	mylog.Check(scanner.Err())

	if len(rootHash) != 64 {
		return "", fmt.Errorf("internal error: unexpected root hash length")
	}

	return rootHash, nil
}

func verityVersion() (major, minor, patch int, err error) {
	output, stderr := mylog.Check3(osutil.RunSplitOutput("veritysetup", "--version"))

	exp := regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)
	match := exp.FindStringSubmatch(string(output))
	if len(match) != 4 {
		return -1, -1, -1, fmt.Errorf("cannot detect veritysetup version from: %s", string(output))
	}
	major = mylog.Check2(strconv.Atoi(match[1]))

	minor = mylog.Check2(strconv.Atoi(match[2]))

	patch = mylog.Check2(strconv.Atoi(match[3]))

	return major, minor, patch, nil
}

func shouldApplyNewFileWorkaroundForOlderThan204() (bool, error) {
	major, minor, patch := mylog.Check4(verityVersion())

	// From version 2.0.4 we don't need this anymore
	if major > 2 || (major == 2 && (minor > 0 || patch >= 4)) {
		return false, nil
	}
	return true, nil
}

// Format runs "veritysetup format" and returns an Info struct which includes the
// root hash. "veritysetup format" calculates the hash verification data for
// dataDevice and stores them in hashDevice. The root hash is retrieved from
// the command's stdout.
func Format(dataDevice string, hashDevice string) (*Info, error) {
	// In older versions of cryptsetup there is a bug when cryptsetup writes
	// its superblock header, and there isn't already preallocated space.
	// Fixed in commit dc852a100f8e640dfdf4f6aeb86e129100653673 which is version 2.0.4
	deploy := mylog.Check2(shouldApplyNewFileWorkaroundForOlderThan204())

	output, stderr := mylog.Check3(osutil.RunSplitOutput("veritysetup", "format", dataDevice, hashDevice))

	logger.Debugf("cmd: 'veritysetup format %s %s':\n%s", dataDevice, hashDevice, string(output))

	rootHash := mylog.Check2(getRootHashFromOutput(output))

	return &Info{RootHash: rootHash}, nil
}
