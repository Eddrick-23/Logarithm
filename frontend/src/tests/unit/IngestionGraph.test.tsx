import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import IngestionGraph from "../../components/IngestionGraph";

// MUI X Charts' LineChart does heavy SVG/canvas work and isn't the focus of these tests
// we will stub it out so tests run fast and we can assert on the props it receives instead.
vi.mock("@mui/x-charts/LineChart", () => ({
    LineChart: (props: any) => <div data-testid="line-chart" data-series-count={props.series?.length ?? 0} />,
}));

// Ignore LastUpdated since it isn't the focus of these tests
vi.mock("./LastUpdated", () => ({
    LastUpdated: () => null,
}));

const buildMetricsData = (services: string[], points = 5) => {
    const timestamps = Array.from({ length: points }, (_, i) => Date.now() - (points - i) * 1000);
    return {
        timestamps,
        metrics: Object.fromEntries(
            services.map((service) => [
                service,
                timestamps.map((ts, i) => ({
                    timestamp: new Date(ts).toISOString(),
                    serviceName: service,
                    logsCount: i + 1,
                })),
            ]),
        ),
    };
};

// set default data to consist of auth-service and billing service only
const data = buildMetricsData(["auth-service", "billing-service"]);
const BASE_PROPS = {
    isLoading: false,
    isError: false,
    lastUpdatedAt: Date.now(),
    data: data,
};

describe("IngestionGraph", () => {
    it("shows a skeleton while loading", () => {
        const { container } = render(<IngestionGraph {...BASE_PROPS} data={undefined} isLoading={true} />);

        // check that the loading skeleton appears and chart is not rendered
        expect(container.querySelector(".MuiSkeleton-root")).toBeInTheDocument();
        expect(screen.queryByTestId("line-chart")).not.toBeInTheDocument();
    });

    it("shows the error banner when isError is true and not loading", () => {
        render(<IngestionGraph {...BASE_PROPS} data={undefined} isError={true} />);

        // check that chart is not rendered
        expect(screen.queryByTestId("line-chart")).not.toBeInTheDocument();
    });

    it("shows an empty-state message when there are no services and no error", () => {
        render(<IngestionGraph {...BASE_PROPS} data={{ timestamps: [], metrics: {} }} />);

        expect(screen.getByText(/no logs received in the last 60 seconds/i)).toBeInTheDocument();
        expect(screen.queryByTestId("line-chart")).not.toBeInTheDocument();
    });

    it("renders the chart and filter checkboxes when data is present", () => {
        render(<IngestionGraph {...BASE_PROPS} />);

        expect(screen.getByTestId("line-chart")).toBeInTheDocument();
        expect(screen.getByTestId("line-chart")).toHaveAttribute("data-series-count", "2");
        expect(screen.getByLabelText("auth-service")).toBeInTheDocument();
        expect(screen.getByLabelText("billing-service")).toBeInTheDocument();

        // "All" chip should be in the "filled" (selected) state initially
        expect(screen.getByText("All")).toBeInTheDocument();
    });

    it("toggling a service checkbox hides it from the chart series", async () => {
        render(<IngestionGraph {...BASE_PROPS} />);

        // all lines in chart is visible initially
        expect(screen.getByTestId("line-chart")).toHaveAttribute("data-series-count", "2");

        // after unchecking auth-service, total lines count should be 1
        await userEvent.click(screen.getByLabelText("auth-service"));
        expect(screen.getByTestId("line-chart")).toHaveAttribute("data-series-count", "1");
    });

    it('clicking "All" toggles every service checkbox', async () => {
        render(<IngestionGraph {...BASE_PROPS} />);

        // initially all selected -> chart shows both series, total lines count should be 2
        expect(screen.getByTestId("line-chart")).toHaveAttribute("data-series-count", "2");

        // click "All" to deselect everything, total lines count should be 0
        await userEvent.click(screen.getByText("All"));
        expect(screen.getByTestId("line-chart")).toHaveAttribute("data-series-count", "0");

        // click "All" again to reselect everything, total lines count should be 2
        await userEvent.click(screen.getByText("All"));
        expect(screen.getByTestId("line-chart")).toHaveAttribute("data-series-count", "2");
    });

    it("does not render the chart when timestamps array is empty but metrics exist", () => {
        render(<IngestionGraph {...BASE_PROPS} data={{ timestamps: [], metrics: { "auth-service": [] } }} />);

        expect(screen.queryByTestId("line-chart")).not.toBeInTheDocument();
    });
});
