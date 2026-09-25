import { execSync } from 'child_process';
import { readFileSync } from 'fs';
import { resolve } from 'path';

const versionPath = resolve(process.cwd(), '../cmd/liki-web/VERSION');
const calver = readFileSync(versionPath, 'utf-8').trim();

// 强缓存失效：CalVer 仅日期，同日多次改动版本不变；而 /css/*、/js/* 是 immutable 长缓存，
// 浏览器/中间层会继续用旧文件。追加 git 短 SHA——每次提交即换版本，强制重新拉取。
let sha = '';
try {
  sha = execSync('git rev-parse --short HEAD', { encoding: 'utf-8' }).trim();
} catch {
  // 无 .git（如 tar 打包构建）时退化为纯 CalVer。
}
export const v = sha ? `${calver}-${sha}` : calver;
