import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/live-tail")({
    component: LiveTailPage,
});

function LiveTailPage() {
    return <div>Hello "/live-tail"!</div>;
}
