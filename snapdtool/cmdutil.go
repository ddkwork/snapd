// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2019-2020 Canonical Ltd
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

package snapdtool

import (
	"bufio"
	"bytes"
	"debug/elf"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ddkwork/golibrary/mylog"
	"github.com/snapcore/snapd/dirs"
	"github.com/snapcore/snapd/osutil"
)

func elfInterp(cmd string) (string, error) {
	el := mylog.Check2(elf.Open(cmd))

	defer el.Close()

	for _, prog := range el.Progs {
		if prog.Type == elf.PT_INTERP {
			r := prog.Open()
			interp := mylog.Check2(io.ReadAll(r))

			return string(bytes.Trim(interp, "\x00")), nil
		}
	}

	return "", fmt.Errorf("cannot find PT_INTERP header")
}

func parseLdSoConf(root string, confPath string) []string {
	f := mylog.Check2(os.Open(filepath.Join(root, confPath)))

	defer f.Close()

	var out []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "#"):
			// nothing
		case strings.TrimSpace(line) == "":
			// nothing
		case strings.HasPrefix(line, "include "):
			l := strings.SplitN(line, "include ", 2)
			files := mylog.Check2(filepath.Glob(filepath.Join(root, l[1])))

			for _, f := range files {
				out = append(out, parseLdSoConf(root, f[len(root):])...)
			}
		default:
			out = append(out, filepath.Join(root, line))
		}

	}
	mylog.Check(scanner.Err())

	return out
}

// CommandFromSystemSnap runs a command from the snapd/core snap
// using the proper interpreter and library paths.
//
// At the moment it can only run ELF files, expects a standard ld.so
// interpreter, and can't handle RPATH.
func CommandFromSystemSnap(name string, cmdArgs ...string) (*exec.Cmd, error) {
	from := "snapd"
	root := filepath.Join(dirs.SnapMountDir, "/snapd/current")
	if !osutil.FileExists(root) {
		from = "core"
		root = filepath.Join(dirs.SnapMountDir, "/core/current")
	}

	cmdPath := filepath.Join(root, name)
	interp := mylog.Check2(elfInterp(cmdPath))

	coreLdSo := filepath.Join(root, interp)
	// we cannot use EvalSymlink here because we need to resolve
	// relative and an absolute symlinks differently. A absolute
	// symlink is relative to root of the snapd/core snap.
	seen := map[string]bool{}
	for osutil.IsSymlink(coreLdSo) {
		link := mylog.Check2(os.Readlink(coreLdSo))

		if filepath.IsAbs(link) {
			coreLdSo = filepath.Join(root, link)
		} else {
			coreLdSo = filepath.Join(filepath.Dir(coreLdSo), link)
		}
		if seen[coreLdSo] {
			return nil, fmt.Errorf("cannot run command from %s: symlink cycle found", from)
		}
		seen[coreLdSo] = true
	}

	ldLibraryPathForCore := parseLdSoConf(root, "/etc/ld.so.conf")

	ldSoArgs := []string{"--library-path", strings.Join(ldLibraryPathForCore, ":"), cmdPath}
	allArgs := append(ldSoArgs, cmdArgs...)
	return exec.Command(coreLdSo, allArgs...), nil
}
