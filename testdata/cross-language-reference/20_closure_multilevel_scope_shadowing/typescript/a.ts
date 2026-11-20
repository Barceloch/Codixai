function target(): void {}
function outer(): void {
  const target = (): void => {};
  function mid(): void {
    const target = (): void => {};
    function inner(): void { target(); }
    inner();
  }
  mid();
}
