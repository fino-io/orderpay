import { useState } from "react";
import { Outlet, useOutletContext } from "react-router-dom";
import { AccessBoundary } from "@/lib/access";

type Position = { status: number; tokens: string[]; index: number; scroll: number };
type OrderWorkspace = { position: Position; setPosition: (position: Position) => void; resetList: () => void };
export const firstPage = (status: number): Position => ({ status, tokens: [""], index: 0, scroll: 0 });

export default function OrdersLayout() {
  const [position, setPosition] = useState(firstPage(0));
  return (
    <AccessBoundary mode="merchant">
      <Outlet
        context={
          { position, setPosition, resetList: () => setPosition(firstPage(position.status)) } satisfies OrderWorkspace
        }
      />
    </AccessBoundary>
  );
}

export const useOrderWorkspace = () => useOutletContext<OrderWorkspace>();
