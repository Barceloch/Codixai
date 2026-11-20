function target() {}
function outer() {
  function inner() {
    const target = () => {};
    target();
  }
  inner();
}
