import { execSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

// 版本文件在仓库根的 cmd/liki-web/VERSION。Astro 的静态构建把 frontmatter 打进
// 预渲染 bundle，import.meta.url 不再指向源码位置，所以按构建 cwd 解析（web/）。
const versionPath = resolve(process.cwd(), '../cmd/liki-web/VERSION');
const calver = readFileSync(versionPath, 'utf-8').trim();

// 缓存失效：Astro 的 /assets/* 带内容哈希文件名，但 /css/* 是原样拷贝的固定名，靠
// must-revalidate + 这里的 ?v= 换缓存 key。CalVer 只精确到日期——同日多次改动版本
// 不变，带同一个 ?v= 的 URL 会被复用。追加 git 短 SHA：每次提交即换版本。
let sha = '';
try {
  sha = execSync('git rev-parse --short HEAD', { encoding: 'utf-8' }).trim();
} catch {
  // 无 .git（如 tar 打包构建）时退化为纯 CalVer。
}
export const v = sha ? `${calver}-${sha}` : calver;
