// The frontend of the packaging spike.
//
// It is plain JavaScript on purpose: no bundler, no framework, no Node.js at
// build time. The Wails v2 runtime injects window.go, which holds the
// methods of the bound Go structs, and window.runtime, which holds the
// runtime API.
//
// Every value that comes from the repository is written with textContent,
// never with innerHTML: the content of the analyzed repository is data, not
// markup.

(function () {
  "use strict";

  // appBinding returns the bound Go struct.
  //
  // Wails v2 exposes it under window.go.<package>.<Struct>. The lookup
  // keeps the method name explicit and tolerates a different package name,
  // so the spike does not depend on the generated wrapper of a Node
  // frontend, which does not exist here.
  function appBinding() {
    var root = window.go || {};
    if (root.ui && root.ui.App && typeof root.ui.App.Audit === "function") {
      return root.ui.App;
    }
    var packages = Object.keys(root);
    for (var i = 0; i < packages.length; i++) {
      var groups = root[packages[i]] || {};
      var structs = Object.keys(groups);
      for (var j = 0; j < structs.length; j++) {
        var candidate = groups[structs[j]];
        if (candidate && typeof candidate.Audit === "function") {
          return candidate;
        }
      }
    }
    return null;
  }

  // waitForBinding resolves the binding, or null after the timeout.
  //
  // The runtime is a script of the page, so the binding exists once the
  // document is complete; the wait keeps a slow start from losing the first
  // interaction.
  function waitForBinding(timeoutMillis) {
    var started = Date.now();
    return new Promise(function (resolve) {
      (function poll() {
        var binding = appBinding();
        if (binding) {
          resolve(binding);
          return;
        }
        if (Date.now() - started >= timeoutMillis) {
          resolve(null);
          return;
        }
        setTimeout(poll, 50);
      })();
    });
  }

  function element(id) {
    return document.getElementById(id);
  }

  function setStatus(text, kind) {
    var status = element("status");
    status.textContent = text;
    status.className = "status" + (kind ? " " + kind : "");
  }

  // summaryRow appends one label and one value to the summary list.
  function summaryRow(list, label, value) {
    var term = document.createElement("dt");
    term.textContent = label;
    var description = document.createElement("dd");
    description.textContent = String(value);
    list.appendChild(term);
    list.appendChild(description);
  }

  function countMap(counts) {
    var keys = Object.keys(counts || {}).sort();
    if (keys.length === 0) {
      return "0";
    }
    return keys
      .map(function (key) {
        return key + ": " + counts[key];
      })
      .join(", ");
  }

  function renderSummary(view) {
    var list = element("summary");
    list.textContent = "";
    var document_ = view.document || {};
    var summary = document_.summary || {};
    summaryRow(list, "Repository", view.root);
    summaryRow(list, "Origin", view.origin + (view.base ? " " + view.base : ""));
    summaryRow(list, "Changed files", view.changed_files);
    summaryRow(list, "Findings", summary.total || 0);
    summaryRow(list, "By severity", countMap(summary.by_severity));
    summaryRow(list, "By epistemic status", countMap(summary.by_epistemic_status));
    summaryRow(list, "By rule", countMap(summary.by_rule));

    var threshold = element("threshold");
    if (view.failed) {
      threshold.textContent =
        view.failing_findings +
        " finding(s) reach the failure threshold (fail_on: " +
        view.fail_on +
        ").";
      threshold.className = "threshold failing";
    } else {
      threshold.textContent =
        "No finding reaches the failure threshold (fail_on: " +
        view.fail_on +
        ").";
      threshold.className = "threshold";
    }
  }

  // renderFinding builds the list item of one finding.
  //
  // The fields are the ones of the versioned document: rule, severity,
  // epistemic status, language, location and message. The interface adds no
  // judgement of its own.
  function renderFinding(finding) {
    var item = document.createElement("li");
    item.className = "finding";

    var head = document.createElement("div");
    head.className = "finding-head";
    var rule = document.createElement("span");
    rule.className = "rule";
    rule.textContent = finding.rule_id;
    var severity = document.createElement("span");
    severity.className = "badge severity-" + finding.severity;
    severity.textContent = finding.severity;
    var epistemic = document.createElement("span");
    epistemic.className = "badge";
    epistemic.textContent = finding.epistemic_status;
    head.appendChild(rule);
    head.appendChild(severity);
    head.appendChild(epistemic);
    if (finding.language) {
      var language = document.createElement("span");
      language.className = "badge";
      language.textContent = finding.language;
      head.appendChild(language);
    }
    item.appendChild(head);

    var location = document.createElement("p");
    location.className = "location";
    var where = finding.location && finding.location.path ? finding.location.path : "(unknown path)";
    if (finding.location && finding.location.start_line) {
      where += ":" + finding.location.start_line;
    }
    location.textContent = where;
    item.appendChild(location);

    var message = document.createElement("p");
    message.className = "message";
    message.textContent = finding.message;
    item.appendChild(message);

    if (finding.recommendation) {
      var recommendation = document.createElement("p");
      recommendation.className = "recommendation";
      recommendation.textContent = finding.recommendation;
      item.appendChild(recommendation);
    }
    return item;
  }

  function renderDocument(view) {
    renderSummary(view);
    var list = element("findings");
    list.textContent = "";
    var findings = (view.document && view.document.findings) || [];
    element("empty").classList.toggle("hidden", findings.length > 0);
    for (var i = 0; i < findings.length; i++) {
      list.appendChild(renderFinding(findings[i]));
    }
    element("result").classList.remove("hidden");
  }

  function selectedOrigin() {
    var selected = document.querySelector('input[name="origin"]:checked');
    return selected ? selected.value : "worktree";
  }

  function setRunning(running) {
    element("run").disabled = running;
  }

  async function runAudit(event) {
    event.preventDefault();
    var binding = await waitForBinding(3000);
    if (!binding) {
      setStatus("The window runtime is not available: start the variant with the ui tag with 'codixai ui'.", "error");
      return;
    }
    var origin = selectedOrigin();
    var request = {
      root: element("root").value,
      worktree: origin === "worktree",
      base: origin === "base" ? element("base").value : "",
    };
    setRunning(true);
    setStatus("Running the audit...", "busy");
    try {
      var view = await binding.Audit(request);
      renderDocument(view);
      setStatus("Audit completed.", "ok");
    } catch (error) {
      var message = error && error.message ? error.message : String(error);
      setStatus(message, "error");
    } finally {
      setRunning(false);
    }
  }

  function syncOrigin() {
    element("base").disabled = selectedOrigin() !== "base";
  }

  async function prefillRoot() {
    var binding = await waitForBinding(3000);
    if (!binding || typeof binding.Root !== "function") {
      return;
    }
    try {
      var root = await binding.Root();
      if (root) {
        element("root").value = root;
      }
    } catch (error) {
      // The field stays empty and the person writes the path.
    }
  }

  document.addEventListener("DOMContentLoaded", function () {
    element("audit-form").addEventListener("submit", runAudit);
    var origins = document.querySelectorAll('input[name="origin"]');
    for (var i = 0; i < origins.length; i++) {
      origins[i].addEventListener("change", syncOrigin);
    }
    syncOrigin();
    prefillRoot();
  });
})();
