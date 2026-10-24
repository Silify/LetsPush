// Empties the build folder but keeps .gitkeep, so Go can always embed it.
import { mkdirSync, readdirSync, rmSync } from 'node:fs';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const dist = fileURLToPath(new URL('../../internal/webui/dist', import.meta.url));

mkdirSync(dist, { recursive: true });
for (const name of readdirSync(dist)) {
  if (name !== '.gitkeep') rmSync(join(dist, name), { recursive: true, force: true });
}
