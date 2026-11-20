function target() {}
function outer() {
  const target = () => {};
  function mid() {
    const target = () => {};
    function inner() { target(); }
    inner();
  }
  mid();
}
