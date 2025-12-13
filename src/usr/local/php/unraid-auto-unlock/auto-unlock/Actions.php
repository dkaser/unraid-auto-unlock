<?php

namespace AutoUnlock;

use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Message\StreamInterface as StreamInterface;
use Symfony\Component\Process\Exception\ProcessFailedException;
use Symfony\Component\Process\Exception\ProcessTimedOutException;
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
    /**
     * Stream a Symfony Process output directly via echo, bypassing Slim's buffering.
     * Uses getIncrementalOutput and disables output buffering for true streaming.
     * MUST call sendStreamHeaders() before calling this function.
     */
    private static function streamProcess(Process $process, string $startMsg = null, string $timeoutMsg = null): int
    {
        if ($startMsg) {
            echo $startMsg;
            flush();
        }
        try {
            $process->start();
            while ($process->isRunning()) {
                $out = $process->getIncrementalOutput();
                $err = $process->getIncrementalErrorOutput();
                if ($out) {
                    echo $out;
                    flush();
                }
                if ($err) {
                    echo $err;
                    flush();
                }
                usleep(100000); // 0.1s
            }
        } catch (ProcessTimedOutException $e) {
            if ($timeoutMsg) {
                echo $timeoutMsg;
                flush();
            }
            return -1;
        }
        // Final output after process ends
        $out = $process->getIncrementalOutput();
        $err = $process->getIncrementalErrorOutput();
        if ($out) {
            echo $out;
            flush();
        }
        if ($err) {
            echo $err;
            flush();
        }
        if ($process->isTerminated() && $process->getExitCode() === 143 && $timeoutMsg) {
            echo $timeoutMsg;
            flush();
        }
        return $process->getExitCode() ?? -1;
    }

    /**
     * Send headers and disable output buffering for streaming responses.
     * Call this before using streamProcess and then exit after streaming.
     */
    private static function sendStreamHeaders(): void
    {
        // Disable all output buffering
        while (@ob_end_flush());
        ini_set('output_buffering', 'off');
        ini_set('zlib.output_compression', 0);

        // Send headers for streaming
        header('Content-Type: text/plain');
        header('Cache-Control: no-cache');
        header('X-Accel-Buffering: no'); // For nginx
        flush();
    }

    public const BIN_PATH = '/usr/local/php/unraid-auto-unlock/bin/autounlock';

    public static function Unlock(Request $request, Response $response): Response
    {
        self::sendStreamHeaders();

        echo "Checking for existing unlock processes\n";
        flush();
        // First, find and kill any running unlock processes
        $findProcess = new Process(['pgrep', '-f', self::BIN_PATH]);
        $findProcess->run();
        if ($findProcess->isSuccessful()) {
            $pids = explode(PHP_EOL, trim($findProcess->getOutput()));
            foreach ($pids as $pid) {
                if (is_numeric($pid)) {
                    echo "Terminating existing unlock process with PID: {$pid}\n";
                    flush();
                    $killProcess = new Process(['kill', $pid]);
                    $killProcess->run();
                }
            }
            sleep(5); // Give some time for processes to terminate
        }

        $findProcess->run();
        if ($findProcess->isSuccessful()) {
            echo "Error: Unable to terminate existing unlock processes.\n";
            flush();
            exit(1);
        }

        $process = new Process([
            self::BIN_PATH,
            'unlock',
            '--pretty'
        ]);
        $process->setTimeout(300);
        $exitCode = self::streamProcess(
            $process,
            "Unlocking Drives\n",
            "Result: TIMEOUT\n"
        );
        if ($exitCode === 0) {
            echo "Result: SUCCESS\n";
        } else {
            echo "Result: FAIL\n";
        }
        flush();
        exit(0);
    }

    public static function Test(Request $request, Response $response): Response
    {
        self::sendStreamHeaders();

        $process = new Process([
            self::BIN_PATH,
            'unlock',
            '--pretty',
            '--debug',
            '--test'
        ]);
        $exitCode = self::streamProcess(
            $process,
            "Testing Configuration\n"
        );
        if ($exitCode === 0) {
            echo "Result: SUCCESS\n";
        } else {
            echo "Result: FAIL\n";
        }
        flush();
        exit(0);
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
        $body = $response->getBody();

        $sharesTotal  = isset($data['shares_total']) ? (int) $data['shares_total'] : 5;
        $sharesUnlock = isset($data['shares_unlock']) ? (int) $data['shares_unlock'] : 3;
        $keyfileData  = isset($data['keyfile_data']) ? (string) $data['keyfile_data'] : null;

        $keyFileParts   = explode(';base64,', $keyfileData ?? '');
        $keyFileContent = end($keyFileParts) ?: '';

        if (empty($keyFileContent)) {
            $body->write("Error: No keyfile provided.");
            return $response->withStatus(400);
        }

        if ($sharesUnlock < 1 || $sharesTotal < 1 || $sharesUnlock > $sharesTotal || $sharesTotal > 100 || $sharesUnlock > 100) {
            $body->write("Error: Invalid share configuration.");
            return $response->withStatus(400);
        }

        $decodedKeyfile = base64_decode($keyFileContent, true);
        if ($decodedKeyfile === false) {
            $body->write("Error: Invalid keyfile encoding.");
            return $response->withStatus(400);
        }

        if (file_put_contents('/root/keyfile', $decodedKeyfile) === false) {
            $body->write("Error: Unable to write temporary keyfile.");
            return $response->withStatus(500);
        }

        self::sendStreamHeaders();
        if ( ! chmod('/root/keyfile', 0600)) {
            echo "Notice: Unable to set permissions on temporary keyfile.";
        }
        $process = null;
        try {
            $process = new Process([
                self::BIN_PATH,
                'setup',
                '--pretty',
                '--shares', $sharesTotal,
                '--threshold', $sharesUnlock
            ]);
            $exitCode = self::streamProcess(
                $process,
                "Initializing...\n"
            );
            if ($exitCode === 0) {
                echo "Result: SUCCESS\n";
            } else {
                echo "Result: FAIL\n";
            }
        } finally {
            // Clean up temporary keyfile if it still exists
            if (file_exists('/root/keyfile')) {
                unlink('/root/keyfile');
            }
        }
        flush();
        exit(0);
    }

    public static function TestPath(Request $request, Response $response): Response
    {
        self::sendStreamHeaders();

        $data     = (array) $request->getParsedBody();
        $testPath = $data['test_path'] ?? '';

        $process = new Process([
            self::BIN_PATH,
            'testpath',
            '--pretty',
            '--debug',
            $testPath
        ]);
        $exitCode = self::streamProcess(
            $process,
            "Testing path: {$testPath}\n"
        );
        if ($exitCode === 0) {
            echo "Result: SUCCESS\n";
        } else {
            echo "Result: FAIL\n";
        }
        flush();
        exit(0);
    }
}
