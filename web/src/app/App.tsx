import { useCallback, useMemo, useState } from "react";
import { ReactFlowProvider } from "@xyflow/react";
import type { Report } from "../report/schema.gen";
import { indexReport, type ViewMode } from "../report/view";
import { Canvas } from "../graph/Canvas";
import { CanvasStateContext } from "../graph/canvasState";
import { selectionState } from "../graph/selection";
import type { CanvasLayout } from "../graph/layout";
import { Inspector } from "../inspector/Inspector";
import { Header } from "./Header";
import { Navigator } from "./Navigator";
import { Footer } from "./Footer";
import { ReportIndexContext } from "./reportContext";
import { useReducedMotion, useThemePreference } from "./preferences";

export function App({ report }: { report: Report }) {
  const index = useMemo(() => indexReport(report), [report]);
  const [view, setView] = useState<ViewMode>(report.mode === "plan" ? "changes" : "after");
  const [selectedId, setSelectedId] = useState<string | undefined>();
  const [focusRequest, setFocusRequest] = useState<{ id: string; seq: number }>();
  const [theme, setTheme] = useThemePreference("system");
  const [reducedMotion, setReducedMotion] = useReducedMotion();

  const canvasState = useMemo(() => selectionState(index, selectedId), [index, selectedId]);
  const selectFromList = useCallback((id: string) => {
    setSelectedId(id);
    setFocusRequest((prev) => ({ id, seq: (prev?.seq ?? 0) + 1 }));
  }, []);
  const recordLayout = useCallback((layout: CanvasLayout) => {
    // Exposed for the performance budget checks in end-to-end tests.
    document.documentElement.dataset.layoutMs = layout.layoutMs.toFixed(1);
    document.documentElement.dataset.layoutNodes = String(layout.nodes.length);
  }, []);

  return (
    <ReportIndexContext.Provider value={index}>
      <div className={`app${reducedMotion ? " motion-reduced" : ""}`}>
        <Header report={report} view={view} onViewChange={setView} theme={theme} onThemeChange={setTheme} />
        <Navigator index={index} selectedId={selectedId} onSelect={selectFromList} />
        <main className="canvas" aria-label="Diagram">
          <ReactFlowProvider>
            <CanvasStateContext.Provider value={canvasState}>
              <Canvas
                index={index}
                view={view}
                focusRequest={focusRequest}
                reducedMotion={reducedMotion}
                onSelect={setSelectedId}
                onLayout={recordLayout}
              />
            </CanvasStateContext.Provider>
          </ReactFlowProvider>
        </main>
        <Inspector index={index} resource={selectedId ? index.resources.get(selectedId) : undefined} onSelect={selectFromList} />
        <Footer report={report} reducedMotion={reducedMotion} onReducedMotionChange={setReducedMotion} />
      </div>
    </ReportIndexContext.Provider>
  );
}
