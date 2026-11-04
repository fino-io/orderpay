import { Navigate } from "react-router-dom";
import { useAccess } from "@/lib/access";

export default function HomePage() {
  const { connection } = useAccess();
  return <Navigate replace to={connection?.mode === "audit" ? "/admin/audit-events" : "/orders"} />;
}
