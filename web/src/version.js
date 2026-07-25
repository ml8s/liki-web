import { readFileSync } from 'fs';
import { resolve } from 'path';

const versionPath = resolve(process.cwd(), '../cmd/liki/VERSION');
export const v = readFileSync(versionPath, 'utf-8').trim();
