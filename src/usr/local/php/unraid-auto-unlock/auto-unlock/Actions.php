<?php

namespace AutoUnlock;

use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Symfony\Component\Process\Exception\ProcessFailedException;
use Symfony\Component\Process\Process;

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

class Actions
{
    public const BIN_PATH = '/usr/local/php/unraid-auto-unlock/bin/autounlock';

    public static function Test(Request $request, Response $response): Response
    {
        $output = array();
        $retval = null;

        $process = new Process([
            self::BIN_PATH,
            'unlock',
            '--pretty',
            '--debug',
            '--test'
        ]);
        $process->run();

        $responseBody = "Testing Configuration" . PHP_EOL;
        if ( ! $process->isSuccessful()) {
            $responseBody .= "Result: FAIL" . PHP_EOL;
        } else {
            $responseBody .= "Result: SUCCESS" . PHP_EOL;
        }

        $responseBody .= PHP_EOL . "Log:" . PHP_EOL . $process->getErrorOutput();

        $response->getBody()->write($responseBody);
        return $response->withHeader('Content-Type', 'text/plain')->withStatus(200);
    }

    public static function Remove(Request $request, Response $response): Response
    {
        Utils::removeConfigFiles();

        return $response
            ->withHeader('Location', '/Tools/AutoUnlock')
            ->withStatus(303);
    }

    public static function Obscure(Request $request, Response $response): Response
    {
        $data       = (array) $request->getParsedBody();
        $inputValue = isset($data['obscure_value']) ? (string) $data['obscure_value'] : '';

        if (empty($inputValue)) {
            $response->getBody()->write("Error: No input value provided.");
            return $response->withHeader('Content-Type', 'text/plain')->withStatus(400);
        }

        $process = new Process([
            self::BIN_PATH,
            'obscure'
        ]);

        $process->setInput($inputValue);
        $process->run();

        if ( ! $process->isSuccessful()) {
            $response->getBody()->write("Error during obscuring process.");
            return $response->withHeader('Content-Type', 'text/plain')->withStatus(500);
        }
        $obscuredValue = trim($process->getOutput());
        $response->getBody()->write($obscuredValue);
        return $response->withHeader('Content-Type', 'text/plain')->withStatus(200);
    }

    public static function Initialize(Request $request, Response $response): Response
    {
        $data = (array) $request->getParsedBody();

        $sharesTotal  = isset($data['shares_total']) ? (int) $data['shares_total'] : 5;
        $sharesUnlock = isset($data['shares_unlock']) ? (int) $data['shares_unlock'] : 3;
        $keyfileData  = isset($data['keyfile_data']) ? (string) $data['keyfile_data'] : null;

        $keyFileParts   = explode(';base64,', $keyfileData ?? '');
        $keyFileContent = end($keyFileParts) ?: '';

        if (empty($keyFileContent)) {
            $response->getBody()->write("Error: No keyfile provided.");
            return $response->withHeader('Content-Type', 'text/plain')->withStatus(400);
        }

        if ($sharesUnlock < 1 || $sharesTotal < 1 || $sharesUnlock > $sharesTotal || $sharesTotal > 100 || $sharesUnlock > 100) {
            $response->getBody()->write("Error: Invalid share configuration.");
            return $response->withHeader('Content-Type', 'text/plain')->withStatus(400);
        }

        $decodedKeyfile = base64_decode($keyFileContent, true);
        if ($decodedKeyfile === false) {
            $response->getBody()->write("Error: Invalid keyfile encoding.");
            return $response->withHeader('Content-Type', 'text/plain')->withStatus(400);
        }

        if (file_put_contents('/root/keyfile', $decodedKeyfile) === false) {
            $response->getBody()->write("Error: Unable to write temporary keyfile.");
            return $response->withHeader('Content-Type', 'text/plain')->withStatus(500);
        }
        @chmod('/root/keyfile', 0600);

        $output = array();
        $retval = null;

        try {
            $process = new Process([
                self::BIN_PATH,
                'setup',
                '--pretty',
                '--shares', $sharesTotal,
                '--threshold', $sharesUnlock
            ]);
            $process->run();

            if ($process->isSuccessful()) {
                $result = $process->getOutput();
            } else {
                $result = "Error during initialization.";
            }
        } finally {
            // Clean up temporary keyfile if it still exists
            if (file_exists('/root/keyfile')) {
                unlink('/root/keyfile');
            }
        }

        $responseBody = $result . PHP_EOL . PHP_EOL . "Log:" . PHP_EOL . $process->getErrorOutput();
        $response->getBody()->write($responseBody);
        return $response->withHeader('Content-Type', 'text/plain')->withStatus(200);
    }

    public static function TestPath(Request $request, Response $response): Response
    {
        $data     = (array) $request->getParsedBody();
        $testPath = $data['test_path'] ?? '';

        $output = array();
        $retval = null;

        $process = new Process([
            self::BIN_PATH,
            'testpath',
            '--pretty',
            '--debug',
            $testPath
        ]);
        $process->run();

        $responseBody = "Testing path: {$testPath}" . PHP_EOL;
        if ( ! $process->isSuccessful()) {
            $responseBody .= "Result: FAIL" . PHP_EOL;
        } else {
            $responseBody .= "Result: SUCCESS" . PHP_EOL;
        }

        $responseBody .= PHP_EOL . "Log:" . PHP_EOL . $process->getErrorOutput();

        $response->getBody()->write($responseBody);
        return $response->withHeader('Content-Type', 'text/plain')->withStatus(200);
    }
}
