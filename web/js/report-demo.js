// report-demo.js — render declarative demo-report JSON blocks.
//
// A data block declares its target:
//   <script type="application/json" data-demo-target="example-id"
//           data-data-id="example-data" set:html={...}></script>
// This script itself carries the same data attributes so one file serves both
// hepan and mingshu demo pages.
(function () {
  'use strict';

  function render(script) {
    var targetId = script.dataset.demoTarget;
    var sourceId = script.dataset.dataId;
    var target = targetId && document.getElementById(targetId);
    var source = sourceId && document.getElementById(sourceId);
    if (!target || !source || !window.renderReport) return;

    try {
      target.innerHTML = window.renderReport(JSON.parse(source.textContent));
    } catch {
      target.textContent = target.getAttribute('data-error') || 'Render error';
    }
  }

  function init() {
    document
      .querySelectorAll('script[data-demo-target][data-data-id]')
      .forEach(render);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
