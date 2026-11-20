function target() {}
function outer() {
  const target = () => {};
  function inner() { target(); }
  inner();
}
