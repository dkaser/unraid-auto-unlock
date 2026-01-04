package constants

/*
	autounlock - Unraid Auto Unlock
	Copyright (C) 2025-2026 Derek Kaser

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import "time"

const (
	ArrayRetryDelay    = 15 * time.Second
	ArrayStatusTimeout = 120 * time.Second
	ArrayTimeout       = 15 * time.Minute
	StartRetryDelay    = 30 * time.Second

	EncryptionKeyBytes = 32
	EncryptionFileMode = 0o600
	MinPaddingLength   = 64
	MaxPaddingLength   = 1048576
	SignatureBytes     = 32
	NonceBytes         = 12

	StateFileMode = 0o600
	StateDirMode  = 0o700

	LockFile     = "/run/autounlock.lock"
	LockFileMode = 0o600
)
