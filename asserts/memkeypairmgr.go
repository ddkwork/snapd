// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2015 Canonical Ltd
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

package asserts

type memoryKeypairManager struct {
	pairs map[string]map[string]PrivateKey
}

// NewMemoryKeypairMananager creates a new key pair manager with a memory backstore.
func NewMemoryKeypairMananager() KeypairManager {
	return memoryKeypairManager{
		pairs: make(map[string]map[string]PrivateKey),
	}
}

func (mskm memoryKeypairManager) Put(authorityID string, privKey PrivateKey) error {
	keyID := privKey.PublicKey().ID()
	perAuthID := mskm.pairs[authorityID]
	if perAuthID == nil {
		perAuthID = make(map[string]PrivateKey)
		mskm.pairs[authorityID] = perAuthID
	} else if perAuthID[keyID] != nil {
		return errKeypairAlreadyExists
	}
	perAuthID[keyID] = privKey
	return nil
}

func (mskm memoryKeypairManager) Get(authorityID, keyID string) (PrivateKey, error) {
	privKey := mskm.pairs[authorityID][keyID]
	if privKey == nil {
		return nil, errKeypairNotFound
	}
	return privKey, nil
}
