import {
  Admin,
  Resource,
} from "react-admin";
import { GasMeterReadingList } from "./GasMeterReadings";
import { GasMeterReadingShow } from "./GasMeterReadingShow";
import { Layout } from "./Layout";
import { dataProvider } from "./dataProvider";
import { authProvider } from "./authProvider";

export const App = () => (
  <Admin
    layout={Layout}
    dataProvider={dataProvider}
    authProvider={authProvider}
  >
    <Resource
      options={{ label: "Gas Meter Readings" }}
      name="gasmeterreadings"
      list={GasMeterReadingList}
      show={GasMeterReadingShow}
    />
  </Admin>
);
