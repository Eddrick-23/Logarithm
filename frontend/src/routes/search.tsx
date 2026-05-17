import { createFileRoute } from "@tanstack/react-router";
import Search from "../pages/Search";

export const Route = createFileRoute("/search")({
    component: SearchPage,
});

function SearchPage() {
    return <Search />;
}
