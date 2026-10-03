import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "@xyflow/react/dist/base.css";
import "./theme/tokens.css";
import "./app/app.css";
import "./graph/graph.css";
import "./inspector/inspector.css";
import { App } from "./app/App";
import { readEmbeddedReport, readExploreConfig, type LoadResult } from "./report/load";

async function loadReport(): Promise<LoadResult> {
  const embedded = readEmbeddedReport(document);
  if (embedded) return embedded;
  if (import.meta.env.DEV) {
    // Development only: production builds never include report data.
    const sample = await import("../../testdata/golden/terraform-1.16-plan.json?raw");
    const { parseReport } = await import("./report/load");
    return parseReport(sample.default);
  }
  return { ok: false, message: "This page does not contain a tfviz report." };
}

const container = document.getElementById("root");
if (container) {
  loadReport().then((result) => {
    // Clear the static, no-JavaScript fallback once the interactive view is ready.
    document.getElementById("static-summary")?.remove();
    const root = createRoot(container);
    root.render(
      <StrictMode>
        {result.ok ? (
          <App report={result.report} explore={readExploreConfig(document)} />
        ) : (
          <div className="load-error" role="alert">
            <h1>The report could not be opened</h1>
            <p>{result.message}</p>
          </div>
        )}
      </StrictMode>,
    );
  });
}
