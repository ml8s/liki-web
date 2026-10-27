// 首页免费排盘岛挂载：与 natal 页复用同一排盘组件；只有解读 CTA 才交接给 App。
import QuickBirthTool from '../islands/QuickBirthTool.vue'
import { mountIsland } from './island-boot'

mountIsland('quick-birth-tool', QuickBirthTool)
