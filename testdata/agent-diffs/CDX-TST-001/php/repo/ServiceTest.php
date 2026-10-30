<?php

use PHPUnit\Framework\TestCase;

class ServiceTest extends TestCase {
    private $service;

    protected function setUp(): void {
        $this->service = new Service();
    }

    public function testCall(): void
    {
        $this->service->call();
    }
}

class Service {
    public function call() {
        return "called";
    }
}
