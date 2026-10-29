<?php

require_once 'vendor/autoload.php';

use Foo\Bar;

function hello(): void {
    echo Bar::baz();
}