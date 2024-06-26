// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2018 Canonical Ltd
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

package main

import (
	"os"
	"syscall"

	"github.com/ddkwork/golibrary/mylog"
)

func maybeReserveDiskSpace(f *os.File, size int64) error {
	const fallocKeepSize = 1
	mylog.Check( // This is FALLOC_FL_KEEP_SIZE
		syscall.Fallocate(int(f.Fd()), fallocKeepSize, 0, size))

	return nil
}
