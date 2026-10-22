// 首页免费排盘岛挂载：复用 BirthFields，并将完整出生草稿交接给 App 建档。
import QuickBirthTool from '../islands/QuickBirthTool.vue'
import { mountIsland } from './island-boot'

mountIsland('quick-birth-tool', QuickBirthTool)
