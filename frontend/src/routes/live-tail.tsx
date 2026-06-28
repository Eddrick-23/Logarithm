import { createFileRoute } from "@tanstack/react-router";
import LiveTail from "../pages/LiveTail";

export const Route = createFileRoute("/live-tail")({
    component: LiveTailPage,
});

function LiveTailPage() {
    return <LiveTail />;
}
