// in src/users.tsx
import {
  Show,
  SimpleShowLayout,
  TextField,
  RichTextField,
  ImageField,
  DateField,
} from "react-admin";

export const GasMeterReadingShow = () => (
  <Show>
    <SimpleShowLayout>
      <TextField source="id" />
      <TextField source="ocr_data" />
      <RichTextField source="date" />
      <ImageField source="image_data" />
      <DateField label="Publication date" source="date" />
    </SimpleShowLayout>
  </Show>
);
