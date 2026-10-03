import { createContext, useContext } from "react";
import type { ReportIndex } from "../report/view";

export const ReportIndexContext = createContext<ReportIndex | undefined>(undefined);

export function useReportIndex(): ReportIndex {
  const index = useContext(ReportIndexContext);
  if (!index) throw new Error("useReportIndex must be used inside ReportIndexContext");
  return index;
}
