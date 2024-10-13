import { useState, useEffect } from "react";
import {
  List,
  TextField,
  ImageField,
  DateField,
  TopToolbar,
  SelectColumnsButton,
  DatagridConfigurable,
  Button,
  FunctionField,
  NumberField,
  DeleteButton,
} from "react-admin";
import { Line } from "react-chartjs-2";
import "chart.js/auto";
import zoomPlugin from "chartjs-plugin-zoom";
import { Chart } from "chart.js";

// Import the date adapter for Chart.js time scaling
import "chartjs-adapter-date-fns";

// Register the zoom plugin with Chart.js
Chart.register(zoomPlugin);

const PostListActions = () => (
  <TopToolbar>
    <SelectColumnsButton />
  </TopToolbar>
);

export const GasMeterReadingList = () => {
  const [chartData, setChartData] = useState<{
    labels: string[];
    datasets: {
      label: string;
      data: number[];
      borderColor: string;
      backgroundColor: string;
      fill: boolean;
    }[];
  }>({
    labels: [],
    datasets: [],
  });

  // Fetch the gas meter reading data for the chart
  useEffect(() => {
    fetch(
      `${import.meta.env.VITE_BACKEND_URL}/gasmeterreadings?_sort=date&_order=DESC&_end=800`,
    )
      .then((response) => response.json())
      .then((data) => {
        // Prepare chart data from the response
        const labels = data.map(
          (reading: { date: string | number | Date }) =>
            new Date(reading.date).toISOString(), // ISO format for accurate time display
        );
        const deltaCounters = data.map(
          (reading: { daily_average: any }) => reading.daily_average,
        );

        setChartData({
          labels,
          datasets: [
            {
              label: "Delta Counter Over Time",
              data: deltaCounters,
              borderColor: "rgba(70, 69, 19, 1)",
              backgroundColor: "rgba(7, 1, 19, 0.4)",
              fill: true,
            },
          ],
        });
      });
  }, []); // Run only once on mount

  // Chart options with zoom and pan enabled
  const chartOptions = {
    responsive: true,
    type: "bar",
    maintainAspectRatio: false,
    stepped: true,
    animations: {
      y: {
        easing: "easeInOutElastic",
        from: (ctx: { type: string; mode: string; dropped: boolean; }) => {
          if (ctx.type === "data") {
            if (ctx.mode === "default" && !ctx.dropped) {
              ctx.dropped = true;
              return 0;
            }
          }
        },
      },
    },
    scales: {
      x: {
        type: "time", // Use time scale for x-axis
        time: {
          unit: "month", // Adapt this to your needs (e.g., "hour", "day", etc.)
          displayFormats: {
            day: "MMM dd yy", // Display format for "day"
          },
        },
      },
    },
    plugins: {
      zoom: {
        pan: {
          enabled: true,
          mode: "x",
        },
        zoom: {
          wheel: {
            enabled: true,
          },
          drag: {
            enabled: true,
          },
          pinch: {
            enabled: false,
          },
          mode: "x",
        },
      },
    },
  };

  return (
    <List actions={<PostListActions />}>
      {/* Button to trigger gas reading */}
      <Button
        label="Schedule Gas Reading"
        onClick={() =>
          fetch(`${import.meta.env.VITE_BACKEND_URL}/fetchwithnew`).then(() => {
            window.location.reload();
          })
        }
      />
      {/* Render Chart.js Line Chart with zoom and pan options */}
      {chartData && (
        <div style={{ width: "100%", height: "400px", marginBottom: "20px" }}>
          <Line data={chartData} options={chartOptions} />
        </div>
      )}

      {/* Datagrid with Gas Meter Readings */}
      <DatagridConfigurable>
        <DeleteButton />
        <DateField showTime={true} source="date" />
        <NumberField source="measurement" />
        <FunctionField
          label="Delta"
          render={(record) => {
            const value = parseFloat(record.delta_days);
            const days = Math.floor(value);
            const hours = Math.floor((value - days) * 24);
            const minutes = Math.round(((value - days) * 24 - hours) * 60);
            const duration = [
              days ? `${days} days` : "",
              hours ? `${hours}h` : "",
              minutes ? `${minutes}m` : "",
            ]
              .filter((part) => part)
              .join(", ");
            return duration;
          }}
        />
        <NumberField source="delta_counter" />
        <FunctionField
          label="m3 per day"
          render={(record) => {
            const value1 = parseFloat(record.delta_days);
            const value2 = parseFloat(record.delta_counter);
            return (value2 / value1).toFixed(2);
          }}
        />
        <NumberField source="daily_average" />
        <FunctionField
          label="Euro per day"
          render={(record) => {
            const value1 = parseFloat(record.delta_days);
            const value2 = parseFloat(record.delta_counter);
            return ((value2 / value1) * 1.55).toFixed(2) + " Euro";
          }}
        />
        <ImageField source="image_data" />
        <TextField source="brightness" />
      </DatagridConfigurable>
    </List>
  );
};
