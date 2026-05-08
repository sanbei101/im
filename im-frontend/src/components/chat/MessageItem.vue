<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { getSDK } from '@/lib/sdk'
import type { ChatMessage } from '@/composables/useChat'
import { prepareWithSegments, layout, walkLineRanges } from '@chenglou/pretext'

const props = defineProps<{
  message: ChatMessage
}>()

const isSelf = computed(() => props.message.sender_id === getSDK().getCurrentUser()?.user_id)

const displayTime = computed(() => {
  const time = props.message.server_time
  if (!time) return ''
  const date = new Date(time / 1000)
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
})

const textContent = computed(() => {
  if (props.message.msg_type === 'text') {
    return (props.message.payload as { text: string }).text
  }
  return ''
})

const bubbleRef = ref<HTMLElement | null>(null)
const tightWidth = ref<number | null>(null)

const layoutMetrics = ref({
  font: '',
  lineHeight: 20,
  paddingH: 24
})

const extractStyles = () => {
  if (!bubbleRef.value) return false
  
  const style = window.getComputedStyle(bubbleRef.value)
  
  const fontWeight = style.fontWeight || '400'
  const fontSize = style.fontSize || '14px'
  const fontFamily = style.fontFamily || 'sans-serif'
  layoutMetrics.value.font = `${fontWeight} ${fontSize} ${fontFamily}`
  
  if (style.lineHeight === 'normal') {
    layoutMetrics.value.lineHeight = parseFloat(fontSize) * 1.5
  } else {
    layoutMetrics.value.lineHeight = parseFloat(style.lineHeight)
  }

  layoutMetrics.value.paddingH = parseFloat(style.paddingLeft) + parseFloat(style.paddingRight)
  
  return true
}

const calculateTightWidth = () => {
  if (props.message.msg_type !== 'text' || !textContent.value || !bubbleRef.value) {
    tightWidth.value = null
    return
  }

  if (!layoutMetrics.value.font) {
    if (!extractStyles()) return
  }

  const screenWidth = window.innerWidth
  const maxBubbleWidth = screenWidth * 0.75 
  const maxContentWidth = Math.floor(maxBubbleWidth - layoutMetrics.value.paddingH)

  const prepared = prepareWithSegments(textContent.value, layoutMetrics.value.font, { whiteSpace: 'pre-wrap' })

  let maxLineWidth = 0
  const initialLineCount = walkLineRanges(prepared, maxContentWidth, line => {
    if (line.width > maxLineWidth) maxLineWidth = line.width
  })

  let lo = 1
  let hi = Math.max(1, Math.ceil(maxContentWidth))

  while (lo < hi) {
    const mid = Math.floor((lo + hi) / 2)
    const midLineCount = layout(prepared, mid, layoutMetrics.value.lineHeight).lineCount
    
    if (midLineCount <= initialLineCount) {
      hi = mid
    } else {
      lo = mid + 1
    }
  }

  tightWidth.value = lo + layoutMetrics.value.paddingH
}

onMounted(() => {
  nextTick(() => {
    calculateTightWidth()
  })
  window.addEventListener('resize', calculateTightWidth)
})

onUnmounted(() => {
  window.removeEventListener('resize', calculateTightWidth)
})

watch(textContent, async () => {
  await nextTick()
  calculateTightWidth()
})
</script>

<template>
  <div class="flex gap-2 py-1" :class="isSelf ? 'flex-row-reverse' : 'flex-row'">
    <div class="h-8 w-8 shrink-0 rounded-full bg-muted flex items-center justify-center text-xs font-medium">
      {{ message.sender_id?.[0]?.toUpperCase() || '?' }}
    </div>

    <div class="flex flex-col max-w-[75%]" :class="isSelf ? 'items-end' : 'items-start'">
      <span class="text-xs text-muted-foreground mb-0.5">{{ displayTime }}</span>

      <div class="relative">
        <div ref="bubbleRef"
             class="px-3 py-2 text-sm leading-relaxed whitespace-pre-wrap break-words" 
             :style="{ width: tightWidth ? `${tightWidth}px` : 'fit-content' }"
             :class="isSelf
               ? 'bg-blue-500 text-white rounded-tl-lg rounded-tr-sm rounded-br-lg rounded-bl-lg'
               : 'bg-white text-gray-800 rounded-tl-sm rounded-tr-lg rounded-br-lg rounded-bl-lg border border-gray-200'">
          {{ textContent }}
          <span v-if="message.status === 'sending'" class="ml-2 text-xs opacity-70">
            发送中...
          </span>
        </div>
      </div>
    </div>
  </div>
</template>