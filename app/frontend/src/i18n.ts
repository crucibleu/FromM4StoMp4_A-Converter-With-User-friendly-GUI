import { computed, ref } from 'vue'

type Dict = Record<string, string>

const zh: Dict = {
  'nav.library': '视频库', 'nav.queue': '转换队列', 'nav.player': '播放器', 'nav.settings': '设置',
  'top.noCache': '未选择缓存目录', 'top.videos': '{n} 个视频 · {size}', 'top.rescan': '重新扫描', 'top.change': '更换目录',

  'lib.search': '搜索标题、UP主、合集…',
  'lib.filter.all': '全部', 'lib.filter.ready': '可转换', 'lib.filter.converted': '已转换',
  'lib.filter.incomplete': '未缓存完成', 'lib.filter.broken': '无法转换', 'lib.filter.danmaku': '有弹幕', 'lib.filter.hd': '1080P+',
  'lib.group.series': '按合集分组', 'lib.group.none': '不分组',
  'lib.sort.series': '按合集', 'lib.sort.size': '按大小', 'lib.sort.duration': '按时长',
  'lib.parts': '{n} 个视频', 'lib.selectGroup': '全选可转换', 'lib.unselectGroup': '取消选择',
  'lib.selected': '已选 {n} 个', 'lib.selectAllReady': '全选可转换', 'lib.clearSel': '清除选择',
  'lib.convertSelected': '转换所选', 'lib.outputTo': '输出到', 'lib.change': '更改',
  'lib.noSpace': '磁盘空间可能不足：需要约 {need}，可用 {free}',
  'lib.empty.title': '选择你的 B 站缓存目录', 'lib.empty.body': '通常是 视频/bilibili。你也可以把文件夹直接拖到窗口里。',
  'lib.empty.pick': '选择文件夹', 'lib.empty.detected': '在这台电脑上找到：',
  'lib.none.title': '这个目录里没有找到缓存视频', 'lib.none.body': '请确认选择的是 B 站客户端的缓存目录，或者包含它的上级目录。',
  'lib.noMatch': '没有符合筛选条件的视频',
  'lib.scanning': '正在扫描缓存目录…', 'lib.found': '找到',

  'status.ready': '可转换', 'status.converted': '已转换', 'status.incomplete': '未缓存完成', 'status.broken': '无法转换',
  'note.missing-audio': '缺少音频文件', 'note.missing-video': '缺少视频文件', 'note.missing-streams': '缺少音视频文件', 'note.encrypted': '文件已加密，无法转换',

  'detail.series': '合集', 'detail.uploader': 'UP主', 'detail.video': '视频', 'detail.audio': '音频', 'detail.size': '大小',
  'detail.duration': '时长', 'detail.danmaku': '弹幕', 'detail.cache': '缓存位置', 'detail.output': '输出文件', 'detail.source': '来源',
  'detail.danmakuCount': '{n} 条', 'detail.none': '无', 'detail.preview': '预览', 'detail.openSystem': '在系统播放器中打开',
  'detail.showCache': '打开缓存文件夹', 'detail.playOutput': '播放已转换的视频', 'detail.convertThis': '转换这个视频',
  'detail.previewFailed': '无法在程序内预览这个视频（{codec}）。',
  'detail.empty': '点击左侧的视频查看详情和预览',
  'source.pc': '电脑客户端', 'source.android': '安卓客户端',

  'out.title': '输出设置', 'out.folder': '输出文件夹', 'out.default': '默认（缓存目录下的 output）', 'out.browse': '浏览…',
  'out.template': '文件命名', 'out.custom': '自定义', 'out.example': '示例', 'out.tokens': '可用占位符：{tokens}，用 / 分隔子文件夹',
  'out.preset.default': '合集-UP主 / 分P', 'out.preset.series': '合集 / 序号-分P', 'out.preset.flat': '全部放在一起', 'out.preset.bvid': '合集 / 分P-BV号',
  'out.danmaku': '生成弹幕文件（.xml 供本程序播放，.ass 供其他播放器）', 'out.download': '缓存里没有弹幕时从网络下载',
  'out.overwrite': '覆盖已存在的同名视频', 'out.save': '保存', 'out.cancel': '取消',

  'queue.title': '转换队列', 'queue.workers': '同时转换', 'queue.cancelAll': '全部取消', 'queue.clear': '清除已完成',
  'queue.openOutput': '打开输出目录', 'queue.done': '已完成', 'queue.speed': '写入速度', 'queue.left': '剩余时间', 'queue.failed': '失败',
  'queue.empty.title': '队列是空的', 'queue.empty.body': '在视频库中选择视频，然后点击「转换所选」。', 'queue.toLibrary': '去视频库',
  'job.queued': '排队中', 'job.prepare': '准备音视频', 'job.merge': '合成音视频', 'job.danmaku': '生成弹幕',
  'job.done': '完成', 'job.skipped': '已存在，跳过', 'job.failed': '失败', 'job.canceled': '已取消',
  'job.cancel': '取消', 'job.retry': '重试', 'job.play': '播放', 'job.reveal': '显示文件',
  'queue.finished': '转换完成：{done} 个成功', 'queue.finishedFailed': '转换完成：{done} 个成功，{failed} 个失败',

  'player.empty.title': '还没有转换好的视频', 'player.empty.body': '输出目录 {dir} 中没有 mp4 文件。',
  'player.search': '搜索已转换的视频…', 'player.danmaku': '弹幕', 'player.on': '开', 'player.off': '关',
  'player.speed': '倍速', 'player.pip': '画中画', 'player.fullscreen': '全屏', 'player.next': '下一个',
  'player.cantPlay': '程序内无法播放这个视频', 'player.cantPlayBody': '这台电脑的系统组件不支持 {codec} 解码。可以用系统播放器打开。',
  'player.openSystem': '用系统播放器打开', 'player.watched': '已看完', 'player.left': '剩 {t}', 'player.playing': '正在播放',
  'player.shortcuts': '空格 播放/暂停 · ←/→ 快退/快进 · ↑/↓ 音量 · F 全屏 · D 弹幕 · N 下一个',
  'player.danmakuSettings': '弹幕设置', 'player.opacity': '不透明度', 'player.density': '密度', 'player.size': '字号',

  'set.title': '设置', 'set.folders': '文件夹', 'set.cacheDir': 'B 站缓存目录', 'set.outputDir': '输出目录',
  'set.conversion': '转换', 'set.workers': '同时转换的视频数', 'set.mp4box': 'MP4Box', 'set.mp4boxBuiltin': '使用内置 MP4Box',
  'set.mp4boxCustom': '使用自己的 MP4Box（GPAC）', 'set.appearance': '外观', 'set.effects': '动画效果',
  'set.effects.full': '完整', 'set.effects.reduced': '精简', 'set.effects.off': '关闭',
  'set.effectsHint': '动画在播放视频和窗口最小化时会自动暂停；画面卡顿时会自动切换为精简。',
  'set.language': '界面语言', 'set.lang.auto': '跟随系统', 'set.player': '播放器', 'set.about': '关于',
  'set.version': '版本 {v}', 'set.terms': '仅转换本人通过哔哩哔哩官方客户端合法缓存的视频，且转换结果严格用于个人备份，绝不传播、分享或商用。',
  'set.saved': '设置已保存',

  'strip.idle': '空闲', 'strip.scanning': '正在扫描', 'strip.running': '{n} 个任务进行中', 'strip.free': '{disk} 可用 {size}',
  'strip.mp4boxBuiltin': 'MP4Box · 内置', 'strip.mp4boxCustom': 'MP4Box · 自定义', 'strip.fxAuto': '画面卡顿，已切换为精简动画',

  'terms.title': '使用条款', 'terms.body': '使用本程序需遵守以下条款：仅转换本人通过哔哩哔哩官方客户端合法缓存的视频，且转换结果严格用于个人备份，绝不传播、分享或商用。',
  'terms.agree': '同意并继续', 'terms.quit': '拒绝并退出',

  'err.no-cache-dir': '还没有选择缓存目录', 'err.cache-dir-missing': '缓存目录不存在，请重新选择',
  'err.mp4box-missing': '设置中的 MP4Box 文件不存在', 'err.generic': '出错了：{msg}',
  'toast.added': '已加入队列：{n} 个视频', 'toast.dropped': '已切换到：{dir}', 'toast.notDir': '请拖入文件夹，而不是文件',
}

const en: Dict = {
  'nav.library': 'Library', 'nav.queue': 'Queue', 'nav.player': 'Player', 'nav.settings': 'Settings',
  'top.noCache': 'No cache folder selected', 'top.videos': '{n} videos · {size}', 'top.rescan': 'Rescan', 'top.change': 'Change folder',

  'lib.search': 'Search titles, uploaders, series…',
  'lib.filter.all': 'All', 'lib.filter.ready': 'Ready', 'lib.filter.converted': 'Converted',
  'lib.filter.incomplete': 'Not finished', 'lib.filter.broken': "Can't convert", 'lib.filter.danmaku': 'Has danmaku', 'lib.filter.hd': '1080P+',
  'lib.group.series': 'Group by series', 'lib.group.none': 'No grouping',
  'lib.sort.series': 'By series', 'lib.sort.size': 'By size', 'lib.sort.duration': 'By length',
  'lib.parts': '{n} videos', 'lib.selectGroup': 'Select ready', 'lib.unselectGroup': 'Deselect',
  'lib.selected': '{n} selected', 'lib.selectAllReady': 'Select all ready', 'lib.clearSel': 'Clear',
  'lib.convertSelected': 'Convert selected', 'lib.outputTo': 'Output', 'lib.change': 'Change',
  'lib.noSpace': 'Disk may be too full: needs about {need}, {free} free',
  'lib.empty.title': 'Choose your Bilibili cache folder', 'lib.empty.body': 'Usually Videos/bilibili. You can also drop the folder onto this window.',
  'lib.empty.pick': 'Choose folder', 'lib.empty.detected': 'Found on this computer:',
  'lib.none.title': 'No cached videos in this folder', 'lib.none.body': "Make sure it is the Bilibili client's cache folder, or a folder that contains it.",
  'lib.noMatch': 'No videos match these filters',
  'lib.scanning': 'Scanning the cache folder…', 'lib.found': 'Found',

  'status.ready': 'Ready', 'status.converted': 'Converted', 'status.incomplete': 'Not finished', 'status.broken': "Can't convert",
  'note.missing-audio': 'Audio file missing', 'note.missing-video': 'Video file missing', 'note.missing-streams': 'Audio and video files missing', 'note.encrypted': 'Encrypted, cannot be converted',

  'detail.series': 'Series', 'detail.uploader': 'Uploader', 'detail.video': 'Video', 'detail.audio': 'Audio', 'detail.size': 'Size',
  'detail.duration': 'Length', 'detail.danmaku': 'Danmaku', 'detail.cache': 'Cache', 'detail.output': 'Output', 'detail.source': 'Source',
  'detail.danmakuCount': '{n}', 'detail.none': 'None', 'detail.preview': 'Preview', 'detail.openSystem': 'Open in system player',
  'detail.showCache': 'Open cache folder', 'detail.playOutput': 'Play converted video', 'detail.convertThis': 'Convert this video',
  'detail.previewFailed': "This video can't be previewed in the app ({codec}).",
  'detail.empty': 'Click a video to see its details and preview it',
  'source.pc': 'Desktop client', 'source.android': 'Android client',

  'out.title': 'Output settings', 'out.folder': 'Output folder', 'out.default': 'Default (output inside the cache folder)', 'out.browse': 'Browse…',
  'out.template': 'File names', 'out.custom': 'Custom', 'out.example': 'Example', 'out.tokens': 'Placeholders: {tokens}. Use / for subfolders',
  'out.preset.default': 'Series-uploader / part', 'out.preset.series': 'Series / no.-part', 'out.preset.flat': 'All in one folder', 'out.preset.bvid': 'Series / part-BV id',
  'out.danmaku': 'Save danmaku (.xml for this app, .ass for other players)', 'out.download': "Download danmaku when the cache doesn't have it",
  'out.overwrite': 'Overwrite videos that already exist', 'out.save': 'Save', 'out.cancel': 'Cancel',

  'queue.title': 'Conversion queue', 'queue.workers': 'At once', 'queue.cancelAll': 'Cancel all', 'queue.clear': 'Clear finished',
  'queue.openOutput': 'Open output folder', 'queue.done': 'Done', 'queue.speed': 'Write speed', 'queue.left': 'Time left', 'queue.failed': 'Failed',
  'queue.empty.title': 'The queue is empty', 'queue.empty.body': 'Select videos in the Library, then click "Convert selected".', 'queue.toLibrary': 'Go to Library',
  'job.queued': 'Waiting', 'job.prepare': 'Preparing streams', 'job.merge': 'Merging audio and video', 'job.danmaku': 'Saving danmaku',
  'job.done': 'Done', 'job.skipped': 'Already exists, skipped', 'job.failed': 'Failed', 'job.canceled': 'Canceled',
  'job.cancel': 'Cancel', 'job.retry': 'Retry', 'job.play': 'Play', 'job.reveal': 'Show file',
  'queue.finished': 'Finished: {done} converted', 'queue.finishedFailed': 'Finished: {done} converted, {failed} failed',

  'player.empty.title': 'No converted videos yet', 'player.empty.body': 'There are no mp4 files in {dir}.',
  'player.search': 'Search converted videos…', 'player.danmaku': 'Danmaku', 'player.on': 'On', 'player.off': 'Off',
  'player.speed': 'Speed', 'player.pip': 'Picture in picture', 'player.fullscreen': 'Fullscreen', 'player.next': 'Next',
  'player.cantPlay': "This video can't play in the app", 'player.cantPlayBody': "This computer's system components can't decode {codec}. You can open it in your system player.",
  'player.openSystem': 'Open in system player', 'player.watched': 'Watched', 'player.left': '{t} left', 'player.playing': 'Playing',
  'player.shortcuts': 'Space play/pause · ←/→ seek · ↑/↓ volume · F fullscreen · D danmaku · N next',
  'player.danmakuSettings': 'Danmaku settings', 'player.opacity': 'Opacity', 'player.density': 'Density', 'player.size': 'Size',

  'set.title': 'Settings', 'set.folders': 'Folders', 'set.cacheDir': 'Bilibili cache folder', 'set.outputDir': 'Output folder',
  'set.conversion': 'Conversion', 'set.workers': 'Videos converted at once', 'set.mp4box': 'MP4Box', 'set.mp4boxBuiltin': 'Use the built-in MP4Box',
  'set.mp4boxCustom': 'Use my own MP4Box (GPAC)', 'set.appearance': 'Appearance', 'set.effects': 'Animation',
  'set.effects.full': 'Full', 'set.effects.reduced': 'Reduced', 'set.effects.off': 'Off',
  'set.effectsHint': 'Animation pauses while a video plays and when the window is minimised, and drops to Reduced automatically if it gets choppy.',
  'set.language': 'Language', 'set.lang.auto': 'Follow system', 'set.player': 'Player', 'set.about': 'About',
  'set.version': 'Version {v}', 'set.terms': 'Only convert videos you cached yourself with the official Bilibili client, for personal backup only. Do not share, distribute or use them commercially.',
  'set.saved': 'Settings saved',

  'strip.idle': 'Idle', 'strip.scanning': 'Scanning', 'strip.running': '{n} running', 'strip.free': '{size} free on {disk}',
  'strip.mp4boxBuiltin': 'MP4Box · built-in', 'strip.mp4boxCustom': 'MP4Box · custom', 'strip.fxAuto': 'Animation was choppy, switched to Reduced',

  'terms.title': 'Terms of use', 'terms.body': 'By using this program you agree to only convert videos you cached yourself with the official Bilibili client, and to use the results strictly as a personal backup. Do not share, distribute or use them commercially.',
  'terms.agree': 'Agree and continue', 'terms.quit': 'Decline and quit',

  'err.no-cache-dir': 'No cache folder selected yet', 'err.cache-dir-missing': "The cache folder doesn't exist any more. Choose it again.",
  'err.mp4box-missing': "The MP4Box file chosen in Settings doesn't exist", 'err.generic': 'Something went wrong: {msg}',
  'toast.added': 'Added {n} videos to the queue', 'toast.dropped': 'Switched to {dir}', 'toast.notDir': 'Drop a folder, not a file',
}

const systemLang = (navigator.language || 'zh').toLowerCase().startsWith('zh') ? 'zh' : 'en'
export const langSetting = ref<'auto' | 'zh' | 'en'>('auto')
export const lang = computed(() => (langSetting.value === 'auto' ? systemLang : langSetting.value))

export function t(key: string, params?: Record<string, string | number>): string {
  const dict = lang.value === 'zh' ? zh : en
  let s = dict[key] ?? en[key] ?? key
  if (params) {
    for (const [k, v] of Object.entries(params)) s = s.split(`{${k}}`).join(String(v))
  }
  return s
}

/** 把 Go 返回的错误转成可读文字 */
export function errText(e: unknown): string {
  const msg = String((e as any)?.message ?? e)
  const key = `err.${msg}`
  const s = t(key)
  return s === key ? t('err.generic', { msg }) : s
}
