<?php

namespace AutoUnlock;

use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Slim\Factory\AppFactory;
use Slim\Middleware\OutputBufferingMiddleware;
use Slim\Psr7\Factory\StreamFactory;

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

$streamFactory = new StreamFactory();
$app->add(new OutputBufferingMiddleware($streamFactory, OutputBufferingMiddleware::APPEND));
$errorMiddleware = $app->addErrorMiddleware(true, true, true);

$app->post("{$prefix}/remove", function (Request $request, Response $response, $args) {
    return Actions::Remove($request, $response);
});

$app->post("{$prefix}/obscure", function (Request $request, Response $response, $args) {
    return Actions::Obscure($request, $response);
});

$app->post("{$prefix}/initialize", function (Request $request, Response $response, $args) {
    return Actions::Initialize($request, $response);
});

$app->post("{$prefix}/test", function (Request $request, Response $response, $args) {
    return Actions::Test($request, $response);
});

$app->post("{$prefix}/test_path", function (Request $request, Response $response, $args) {
    return Actions::TestPath($request, $response);
});

$app->post("{$prefix}/open", function (Request $request, Response $response, $args) {
    return Actions::Unlock($request, $response);
});

$app->run();
