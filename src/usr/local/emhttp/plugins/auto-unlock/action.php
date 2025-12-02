<?php

namespace AutoUnlock;

use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Slim\Factory\AppFactory;
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

require_once dirname(__FILE__) . "/include/common.php";

$prefix = "/plugins/auto-unlock/action.php";

if ( ! defined(__NAMESPACE__ . '\PLUGIN_ROOT') || ! defined(__NAMESPACE__ . '\PLUGIN_NAME')) {
    throw new \RuntimeException("Common file not loaded.");
}

$app = AppFactory::create();
$app->addRoutingMiddleware();
$errorMiddleware = $app->addErrorMiddleware(true, true, true);

$app->post("{$prefix}/remove", function (Request $request, Response $response, $args) {
    Utils::removeConfigFiles();

    return $response
        ->withHeader('Location', '/Tools/AutoUnlock')
        ->withStatus(303);
});

$app->post("{$prefix}/initialize", function (Request $request, Response $response, $args) {
    $data = (array) $request->getParsedBody();

    $sharesTotal  = isset($data['shares_total']) ? (int) $data['shares_total'] : 5;
    $sharesUnlock = isset($data['shares_unlock']) ? (int) $data['shares_unlock'] : 3;
    $keyfileData  = isset($data['keyfile_data']) ? $data['keyfile_data'] : null;

    $keyFileParts   = explode(';base64,', $keyfileData);
    $keyFileContent = end($keyFileParts);

    if (empty($keyFileContent)) {
        $response->getBody()->write("Error: No keyfile provided.");
        return $response->withHeader('Content-Type', 'text/plain')->withStatus(400);
    }

    if ($sharesUnlock < 1 || $sharesTotal < 1 || $sharesUnlock > $sharesTotal || $sharesTotal > 100 || $sharesUnlock > 100) {
        $response->getBody()->write("Error: Invalid share configuration.");
        return $response->withHeader('Content-Type', 'text/plain')->withStatus(400);
    }

    file_put_contents('/root/keyfile', base64_decode($keyFileContent));

    $output = array();
    $retval = null;

    $process = new Process([
        '/usr/local/php/unraid-auto-unlock/bin/autounlock',
        '--pretty',
        '--setup',
        '--shares', $sharesTotal,
        '--threshold', $sharesUnlock
    ]);
    $process->run();

    // Clean up temporary keyfile if it still exists
    if (file_exists('/root/keyfile')) {
        unlink('/root/keyfile');
    }

    if ($process->isSuccessful()) {
        $result = $process->getOutput();
    } else {
        $result = "Error during initialization.";
    }

    $responseBody = $result . PHP_EOL . PHP_EOL . "Log:" . PHP_EOL . $process->getErrorOutput();
    $response->getBody()->write($responseBody);
    return $response->withHeader('Content-Type', 'text/plain')->withStatus(200);
});

$app->post("{$prefix}/test", function (Request $request, Response $response, $args) {
    $output = array();
    $retval = null;
    exec("/usr/local/php/unraid-auto-unlock/bin/autounlock --pretty --debug --test 2>&1", $output, $retval);
    $responseBody = "Testing Configuration" . PHP_EOL;
    if ($retval != 0) {
        $responseBody .= "Result: FAIL" . PHP_EOL;
    } else {
        $responseBody .= "Result: SUCCESS" . PHP_EOL;
    }

    $responseBody .= PHP_EOL . "Log:" . PHP_EOL . implode(PHP_EOL, $output);

    $response->getBody()->write($responseBody);
    return $response->withHeader('Content-Type', 'text/plain')->withStatus(200);
});

$app->post("{$prefix}/test_path", function (Request $request, Response $response, $args) {
    $data     = (array) $request->getParsedBody();
    $testPath = $data['test_path'] ?? '';

    $output = array();
    $retval = null;

    $testPath = escapeshellarg($testPath);
    exec("/usr/local/php/unraid-auto-unlock/bin/autounlock --pretty --debug --test-path {$testPath} 2>&1", $output, $retval);

    $responseBody = "Testing path: {$testPath}" . PHP_EOL;
    if ($retval != 0) {
        $responseBody .= "Result: FAIL" . PHP_EOL;
    } else {
        $responseBody .= "Result: SUCCESS" . PHP_EOL;
    }

    $responseBody .= PHP_EOL . "Log:" . PHP_EOL . implode(PHP_EOL, $output);

    $response->getBody()->write($responseBody);
    return $response->withHeader('Content-Type', 'text/plain')->withStatus(200);
});

$app->run();
