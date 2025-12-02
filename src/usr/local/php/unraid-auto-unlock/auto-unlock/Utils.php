<?php

namespace AutoUnlock;

/*
    Copyright (C) 2025  Derek Kaser

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

class Utils extends \EDACerton\PluginUtils\Utils
{
    public const STATE_FILE  = "/boot/config/plugins/auto-unlock/state.json";
    public const ENC_FILE    = "/boot/config/plugins/auto-unlock/unlock.enc";
    public const CONFIG_FILE = "/boot/config/plugins/auto-unlock/config.txt";

    public static function removeConfigFiles(): void
    {
        if (file_exists(self::STATE_FILE)) {
            unlink(self::STATE_FILE);
        }
        if (file_exists(self::ENC_FILE)) {
            unlink(self::ENC_FILE);
        }
    }

    public static function getCsrfToken(): string
    {
        $var_ini = parse_ini_file("/var/local/emhttp/var.ini");
        if ($var_ini === false || !isset($var_ini['csrf_token'])) {
            return '';
        }
        return (string) $var_ini['csrf_token'];
    }
}
