import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/analytics")({
    component: AnalyticsPage,
});

function AnalyticsPage() {
    return <div>Hello "/analytics"!</div>;
}
