const foo = require('./foo');

function hello() {
    console.log(foo.bar());
}

module.exports = { hello };