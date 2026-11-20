function target() {}
function outer() {
  function inner() { target(); }
  var target = () => {};
  inner();
}
