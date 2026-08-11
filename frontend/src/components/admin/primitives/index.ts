export {
  AdminPage,
  Footnote,
  ForbiddenState,
  SectionLabel,
  StaleSessionState,
} from "./page";
export { KpiRow, KpiRowSkeleton, type KpiDef } from "./kpi";
export {
  Field,
  FieldGroup,
  fieldClass,
  SearchInput,
  SelectInput,
  TextArea,
  TextInput,
} from "./controls";
export {
  RecordCard,
  RecordField,
  Table,
  TableFrame,
  TableScroll,
  TableToolbar,
  Td,
  Th,
  Tr,
} from "./table";
export { LoadBar, StaffStatusBadge } from "./indicators";
/*
  StaffCardGrid is deliberately NOT re-exported here. It consumes LoadBar and
  StaffStatusBadge from this barrel, so listing it would make primitives/index
  import a module that imports primitives/index. Callers take it from
  components/admin/staff/staff-card-grid directly, as they do StaffTable.
*/
