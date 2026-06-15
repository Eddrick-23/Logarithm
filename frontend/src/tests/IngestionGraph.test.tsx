import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import IngestionGraph from "../components/IngestionGraph";
import { useIngestionMetrics } from "../hooks/useMetrics";

vi.mock("../hooks/useMetrics", () => ({
    useIngestionMetrics: vi.fn(),
}));

// MUI X Charts' LineChart does heavy SVG/canvas work and isn't the focus of these tests
// we will stub it out so tests run fast and we can assert on the props it receives instead.
vi.mock("@mui/x-charts/LineChart", () => ({
    LineChart: (props: any) => <div data-testid="line-chart" data-series-count={props.series?.length ?? 0} />,
}));

const mockUseIngestionMetrics = useIngestionMetrics as unknown as ReturnType<typeof vi.fn>;

const buildMetricsData = (services: string[], points = 5) => ({
    timestamps: Array.from({ length: points }, (_, i) => Date.now() - (points - i) * 1000),
    metrics: Object.fromEntries(
        services.map((service) => [service, Array.from({ length: points }, (_, i) => ({ logsCount: i + 1 }))]),
    ),
});

describe("IngestionGraph", () => {
    beforeEach(() => {
        mockUseIngestionMetrics.mockReset();
    });

    it("shows a skeleton while loading", () => {
        mockUseIngestionMetrics.mockReturnValue({
            data: undefined,
            isLoading: true,
            isError: false,
            refetch: vi.fn(),
        });

        const { container } = render(<IngestionGraph />);

        expect(container.querySelector(".MuiSkeleton-root")).toBeInTheDocument();
        expect(screen.queryByTestId("line-chart")).not.toBeInTheDocument();
    });

    it("shows the error banner when isError is true and not loading", () => {
        const refetch = vi.fn();
        mockUseIngestionMetrics.mockReturnValue({
            data: undefined,
            isLoading: false,
            isError: true,
            refetch,
        });

        render(<IngestionGraph />);

        expect(screen.getByRole("button", { name: /retry/i })).toBeInTheDocument();
        expect(screen.queryByTestId("line-chart")).not.toBeInTheDocument();
    });

    it("calls refetch when the retry button is clicked", async () => {
        const refetch = vi.fn();
        mockUseIngestionMetrics.mockReturnValue({
            data: undefined,
            isLoading: false,
            isError: true,
            refetch,
        });

        render(<IngestionGraph />);

        await userEvent.click(screen.getByRole("button", { name: /retry/i }));

        expect(refetch).toHaveBeenCalledTimes(1);
    });

    it("shows an empty-state message when there are no services and no error", () => {
        mockUseIngestionMetrics.mockReturnValue({
            data: { timestamps: [], metrics: {} },
            isLoading: false,
            isError: false,
            refetch: vi.fn(),
        });

        render(<IngestionGraph />);

        expect(screen.getByText(/no logs received in the last 60 seconds/i)).toBeInTheDocument();
        expect(screen.queryByTestId("line-chart")).not.toBeInTheDocument();
    });

    it("renders the chart and filter checkboxes when data is present", () => {
        const data = buildMetricsData(["auth-service", "billing-service"]);
        mockUseIngestionMetrics.mockReturnValue({
            data,
            isLoading: false,
            isError: false,
            refetch: vi.fn(),
        });

        render(<IngestionGraph />);

        expect(screen.getByTestId("line-chart")).toBeInTheDocument();
        expect(screen.getByTestId("line-chart")).toHaveAttribute("data-series-count", "2");
        expect(screen.getByLabelText("auth-service")).toBeInTheDocument();
        expect(screen.getByLabelText("billing-service")).toBeInTheDocument();

        // "All" chip should be in the "filled" (selected) state initially
        expect(screen.getByText("All")).toBeInTheDocument();
    });

    it("toggling a service checkbox hides it from the chart series", async () => {
        const data = buildMetricsData(["auth-service", "billing-service"]);
        mockUseIngestionMetrics.mockReturnValue({
            data,
            isLoading: false,
            isError: false,
            refetch: vi.fn(),
        });

        render(<IngestionGraph />);

        expect(screen.getByTestId("line-chart")).toHaveAttribute("data-series-count", "2");

        await userEvent.click(screen.getByLabelText("auth-service"));

        expect(screen.getByTestId("line-chart")).toHaveAttribute("data-series-count", "1");
    });

    it('clicking "All" toggles every service checkbox', async () => {
        const data = buildMetricsData(["auth-service", "billing-service"]);
        mockUseIngestionMetrics.mockReturnValue({
            data,
            isLoading: false,
            isError: false,
            refetch: vi.fn(),
        });

        render(<IngestionGraph />);

        // initially all selected -> chart shows both series
        expect(screen.getByTestId("line-chart")).toHaveAttribute("data-series-count", "2");

        // click "All" to deselect everything
        await userEvent.click(screen.getByText("All"));
        expect(screen.getByTestId("line-chart")).toHaveAttribute("data-series-count", "0");

        // click "All" again to reselect everything
        await userEvent.click(screen.getByText("All"));
        expect(screen.getByTestId("line-chart")).toHaveAttribute("data-series-count", "2");
    });

    it("does not render the chart when timestamps array is empty but metrics exist", () => {
        mockUseIngestionMetrics.mockReturnValue({
            data: { timestamps: [], metrics: { "auth-service": [] } },
            isLoading: false,
            isError: false,
            refetch: vi.fn(),
        });

        render(<IngestionGraph />);

        expect(screen.queryByTestId("line-chart")).not.toBeInTheDocument();
    });
});
