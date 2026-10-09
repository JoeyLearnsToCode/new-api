/**
 * Canvas 渲染引擎 - 基于 HTML Canvas 2D API
 * 负责渲染版本树的节点、连线、标签
 */

import type { Version } from '@/prompt-studio/models/Version';
import { colors } from '@/prompt-studio/styles/tokens';
import {
  buildVersionTree,
  calculateTreeLayout,
  type VersionTreeNode,
} from '@/prompt-studio/utils/tree';

export interface CanvasNode {
  id: string;
  x: number;
  y: number;
  width: number;
  height: number;
  version: Version;
  children: CanvasNode[];
}

export interface CanvasTransform {
  x: number; // 平移 X
  y: number; // 平移 Y
  scale: number; // 缩放比例
}

export class CanvasRenderer {
  private canvas: HTMLCanvasElement;
  private ctx: CanvasRenderingContext2D;
  private transform: CanvasTransform = { x: 0, y: 0, scale: 1 };
  private nodes: CanvasNode[] = [];
  private selectedNodeId: string | null = null;
  private resizeTimer: number | null = null;

  // 🟢 手动微调：修改此数值改变连线圆角的大小 (默认 12)
  private cornerRadius = 12;

  // Theme Colors - initialized with defaults, updated in updateThemeColors
  private themeColors = {
    primary: colors.primary.DEFAULT,
    primaryContainer: colors.background.DEFAULT,
    onPrimary: colors.primary.onPrimary,
    surface: colors.surface.DEFAULT,
    surfaceVariant: colors.surface.variant,
    onSurface: colors.text.light.primary,
    onSurfaceVariant: colors.text.light.secondary,
    outline: colors.border.DEFAULT,
    selectedNode: colors.primary.DEFAULT,
    connection: colors.text.light.muted,
  };

  constructor(canvas: HTMLCanvasElement) {
    this.canvas = canvas;
    const ctx = canvas.getContext('2d');
    if (!ctx) throw new Error('无法获取 Canvas 2D 上下文');
    this.ctx = ctx;

    // Check for dark mode to adjust surface colors
    this.updateThemeColors();
    this.resizeCanvas();
  }

  public updateThemeColors() {
    const isDark = document.documentElement.classList.contains('dark');
    if (isDark) {
      this.themeColors = {
        primary: colors.primary.DEFAULT,
        primaryContainer: colors.surface.onSurface, // Using onSurface as container-like in dark logic for text
        onPrimary: colors.primary.onPrimary,
        surface: colors.surface.dark,
        surfaceVariant: colors.border.dark,
        onSurface: colors.text.dark.primary,
        onSurfaceVariant: colors.text.dark.muted,
        outline: colors.border.dark,
        selectedNode: colors.primary.DEFAULT,
        connection: colors.text.dark.muted,
      };
    } else {
      // Light mode default
      this.themeColors = {
        primary: colors.primary.DEFAULT,
        primaryContainer: colors.background.DEFAULT,
        onPrimary: colors.primary.onPrimary,
        surface: colors.surface.DEFAULT,
        surfaceVariant: colors.surface.variant,
        onSurface: colors.text.light.primary,
        onSurfaceVariant: colors.text.light.secondary,
        outline: colors.border.DEFAULT,
        selectedNode: colors.primary.DEFAULT,
        connection: colors.text.light.muted,
      };
    }
  }

  resizeCanvas() {
    if (this.resizeTimer !== null) {
      return;
    }
    this.performResize();
    this.resizeTimer = window.setTimeout(() => {
      this.resizeTimer = null;
    }, 150);
  }

  private performResize() {
    const dpr = window.devicePixelRatio || 1;
    const parent = this.canvas.parentElement;
    if (!parent) return;

    const rect = parent.getBoundingClientRect();
    const width = rect.width;
    const height = rect.height;

    // 设置实际渲染尺寸（考虑设备像素比）
    this.canvas.width = width * dpr;
    this.canvas.height = height * dpr;
    this.canvas.style.width = `${width}px`;
    this.canvas.style.height = `${height}px`;

    const ctx = this.canvas.getContext('2d');
    if (ctx) {
      this.ctx = ctx;
      // 重新获取 context 后重置 scale
      this.ctx.scale(dpr, dpr);
    }

    this.updateThemeColors(); // Re-check theme on resize/redraw
    this.draw();
  }

  renderTree(versions: Version[]) {
    const roots = buildVersionTree(versions);

    const nodeWidth = 200;
    const nodeHeight = 80;
    const horizontalSpacing = 30;
    const verticalSpacing = 80;

    this.nodes = [];
    let offsetX = 50;

    roots.forEach((root) => {
      const layout = calculateTreeLayout(
        root,
        nodeWidth,
        nodeHeight,
        horizontalSpacing,
        verticalSpacing,
      );
      this.nodes.push(this.convertToCanvasNode(layout, offsetX));

      const treeWidth = this.calculateSubtreeWidth(
        layout,
        nodeWidth,
        horizontalSpacing,
      );
      offsetX += treeWidth + 100;
    });

    this.draw();
  }

  private calculateSubtreeWidth(
    node: VersionTreeNode,
    nodeWidth: number,
    horizontalSpacing: number,
  ): number {
    if (node.children.length === 0) {
      return nodeWidth;
    }
    const childrenWidths = node.children.map((child: VersionTreeNode) =>
      this.calculateSubtreeWidth(child, nodeWidth, horizontalSpacing),
    );
    return (
      childrenWidths.reduce((sum: number, w: number) => sum + w, 0) +
      (node.children.length - 1) * horizontalSpacing
    );
  }

  private convertToCanvasNode(treeNode: any, offsetX: number = 0): CanvasNode {
    const node: CanvasNode = {
      id: treeNode.id,
      x: treeNode.x + offsetX + 50,
      y: treeNode.y + 50,
      width: 200,
      height: 80,
      version: treeNode.version,
      children: [],
    };

    if (treeNode.children) {
      node.children = treeNode.children.map((child: any) =>
        this.convertToCanvasNode(child, offsetX),
      );
    }

    return node;
  }

  public draw() {
    const { ctx, canvas } = this;
    const { x, y, scale } = this.transform;

    // 获取实际渲染尺寸（考虑设备像素比）
    const width = canvas.width / (window.devicePixelRatio || 1);
    const height = canvas.height / (window.devicePixelRatio || 1);

    // 清空画布
    ctx.clearRect(0, 0, width, height);

    ctx.save();
    ctx.translate(x, y);
    ctx.scale(scale, scale);

    // 1. 绘制连线 (递归绘制所有连线)
    this.nodes.forEach((node) => this.drawConnections(node));

    // 2. 绘制所有节点
    // 获取所有扁平化节点以确保所有节点都被绘制
    const allNodes = this.flattenNodes();

    // 排序：将选中的节点放在最后绘制，使其处于最上层
    allNodes.sort((a, b) => {
      if (a.id === this.selectedNodeId) return 1;
      if (b.id === this.selectedNodeId) return -1;
      return 0;
    });

    allNodes.forEach((node) => this.drawNode(node));

    ctx.restore();
  }

  private drawConnections(node: CanvasNode) {
    const { ctx } = this;

    node.children.forEach((child) => {
      const parentCenterX = node.x + node.width / 2;
      const parentBottomY = node.y + node.height;
      const childCenterX = child.x + child.width / 2;
      const childTopY = child.y;

      ctx.beginPath();
      ctx.strokeStyle = this.themeColors.connection;
      ctx.lineWidth = 2;

      // 如果父子节点的 X 坐标几乎相同（垂直对齐），直接画直线
      if (Math.abs(parentCenterX - childCenterX) < 1) {
        ctx.moveTo(parentCenterX, parentBottomY);
        ctx.lineTo(parentCenterX, childTopY);
      } else {
        // 否则画带圆角的折线 (Manhattan routing with rounded corners)
        const midY = parentBottomY + (childTopY - parentBottomY) / 2;

        ctx.moveTo(parentCenterX, parentBottomY);

        // 绘制第一个弯：从父节点底部向下，在 midY 处转向子节点水平方向
        // arcTo 会自动从当前点画一条直线到切点，然后画圆弧
        ctx.arcTo(parentCenterX, midY, childCenterX, midY, this.cornerRadius);

        // 绘制第二个弯：从 midY 水平延伸，在子节点 X 轴处转向向下
        // arcTo 会自动从当前点画一条直线到切点，然后画圆弧
        ctx.arcTo(
          childCenterX,
          midY,
          childCenterX,
          childTopY,
          this.cornerRadius,
        );

        // 最后画直线到子节点顶部
        ctx.lineTo(childCenterX, childTopY);
      }

      ctx.stroke();
      this.drawConnections(child);
    });
  }

  private drawNode(node: CanvasNode) {
    const { ctx } = this;
    const isSelected = node.id === this.selectedNodeId;

    // Background
    if (isSelected) {
      ctx.fillStyle = this.themeColors.selectedNode;
    } else {
      ctx.fillStyle = this.themeColors.surface;
    }

    const isDark = document.documentElement.classList.contains('dark');
    ctx.shadowColor = isDark ? colors.border.dark : colors.border.DEFAULT;
    if (!isSelected) {
      ctx.shadowBlur = 4;
      ctx.shadowOffsetY = 2;
    } else {
      ctx.shadowBlur = 8;
      ctx.shadowOffsetY = 4;
    }

    this.roundRect(ctx, node.x, node.y, node.width, node.height, 8); // 8px radius
    ctx.fill();

    // Reset shadow
    ctx.shadowColor = 'transparent';
    ctx.shadowBlur = 0;
    ctx.shadowOffsetY = 0;

    // Border for unselected nodes
    if (!isSelected) {
      ctx.strokeStyle = this.themeColors.outline;
      ctx.lineWidth = 1;
      ctx.stroke();
    }

    // Text Color
    const textColor = isSelected
      ? this.themeColors.onPrimary
      : this.themeColors.onSurface;

    // Version Name
    let currentY = node.y + 16; // Padding top
    if (node.version.name) {
      ctx.fillStyle = textColor;
      ctx.font = 'bold 14px sans-serif';
      ctx.textBaseline = 'top';

      const displayName =
        node.version.name.length > 20
          ? node.version.name.substring(0, 20) + '...'
          : node.version.name;

      ctx.fillText(displayName, node.x + 12, currentY);
      currentY += 20;
    }

    // Content
    ctx.fillStyle = textColor; // Muted if name exists
    ctx.font = '14px sans-serif';
    ctx.textBaseline = 'top';

    const originalLines = node.version.content
      .split('\n')
      .filter((line) => line.trim() !== '');
    const maxLines = node.version.name ? 2 : 3;
    let currentLineIndex = 0;
    const maxWidth = node.width - 24; // Padding 12px * 2

    for (
      let i = 0;
      i < originalLines.length && currentLineIndex < maxLines;
      i++
    ) {
      const line = originalLines[i];

      const isLastLine = i === originalLines.length - 1;
      const remainingLines = maxLines - currentLineIndex;

      // 检查文本是否超出卡片宽度
      const metrics = ctx.measureText(line);

      if (metrics.width > maxWidth) {
        // 如果是最后一行且还有剩余空间（至少2行），允许换行
        if (isLastLine && remainingLines >= 2) {
          // 将长行分割成多行
          const chars = line.split('');
          let tempLine = '';
          const wrappedLines: string[] = [];

          for (const char of chars) {
            const testLine = tempLine + char;
            if (ctx.measureText(testLine).width > maxWidth) {
              if (tempLine) {
                wrappedLines.push(tempLine);
                tempLine = char;
              } else {
                wrappedLines.push(char);
                tempLine = '';
              }
            } else {
              tempLine = testLine;
            }
          }

          if (tempLine) {
            wrappedLines.push(tempLine);
          }

          // 显示换行后的内容
          const linesToShow = Math.min(wrappedLines.length, remainingLines);
          for (let j = 0; j < linesToShow; j++) {
            let displayText = wrappedLines[j];

            // 如果是最后一行且还有更多内容，添加省略号
            if (j === linesToShow - 1 && wrappedLines.length > linesToShow) {
              let truncatedText = displayText;
              while (
                ctx.measureText(truncatedText + '...').width > maxWidth &&
                truncatedText.length > 0
              ) {
                truncatedText = truncatedText.slice(0, -1);
              }
              displayText = truncatedText + '...';
            }

            ctx.fillText(
              displayText,
              node.x + 8,
              currentY + currentLineIndex * 18,
            );
            currentLineIndex++;
          }
        } else {
          // 其他情况，截断并添加省略号
          let truncatedText = line;
          while (
            ctx.measureText(truncatedText + '...').width > maxWidth &&
            truncatedText.length > 0
          ) {
            truncatedText = truncatedText.slice(0, -1);
          }
          ctx.fillText(
            truncatedText + '...',
            node.x + 8,
            currentY + currentLineIndex * 18,
          );
          currentLineIndex++;
        }
      } else {
        // 文本未超出宽度，直接显示
        ctx.fillText(line, node.x + 8, currentY + currentLineIndex * 18);
        currentLineIndex++;
      }
    }
  }

  /**
   * 绘制圆角矩形
   */
  private roundRect(
    ctx: CanvasRenderingContext2D,
    x: number,
    y: number,
    width: number,
    height: number,
    radius: number,
  ) {
    ctx.beginPath();
    ctx.moveTo(x + radius, y);
    ctx.lineTo(x + width - radius, y);
    ctx.quadraticCurveTo(x + width, y, x + width, y + radius);
    ctx.lineTo(x + width, y + height - radius);
    ctx.quadraticCurveTo(x + width, y + height, x + width - radius, y + height);
    ctx.lineTo(x + radius, y + height);
    ctx.quadraticCurveTo(x, y + height, x, y + height - radius);
    ctx.lineTo(x, y + radius);
    ctx.quadraticCurveTo(x, y, x + radius, y);
    ctx.closePath();
  }

  setTransform(transform: Partial<CanvasTransform>) {
    this.transform = { ...this.transform, ...transform };
    this.draw();
  }

  zoom(delta: number, centerX: number, centerY: number) {
    const newScale = Math.max(0.1, Math.min(3, this.transform.scale + delta));
    const scaleDiff = newScale - this.transform.scale;
    this.transform.x -= centerX * scaleDiff;
    this.transform.y -= centerY * scaleDiff;
    this.transform.scale = newScale;
    this.draw();
  }

  pan(dx: number, dy: number) {
    this.transform.x += dx;
    this.transform.y += dy;
    this.draw();
  }

  selectNode(nodeId: string | null) {
    this.selectedNodeId = nodeId;
    this.draw();
  }

  hitTest(x: number, y: number): string | null {
    const canvasX = (x - this.transform.x) / this.transform.scale;
    const canvasY = (y - this.transform.y) / this.transform.scale;

    for (const node of this.flattenNodes()) {
      if (
        canvasX >= node.x &&
        canvasX <= node.x + node.width &&
        canvasY >= node.y &&
        canvasY <= node.y + node.height
      ) {
        return node.id;
      }
    }
    return null;
  }

  private flattenNodes(): CanvasNode[] {
    const result: CanvasNode[] = [];
    const traverse = (node: CanvasNode) => {
      result.push(node);
      node.children.forEach(traverse);
    };
    this.nodes.forEach(traverse);
    return result;
  }

  resetView() {
    this.transform = { x: 0, y: 0, scale: 1 };
    this.draw();
  }

  centerNode(nodeId: string) {
    const node = this.flattenNodes().find((n) => n.id === nodeId);
    if (!node) return;
    const centerX = this.canvas.width / 2;
    const centerY = this.canvas.height / 2;
    this.transform.x =
      centerX - (node.x + node.width / 2) * this.transform.scale;
    this.transform.y =
      centerY - (node.y + node.height / 2) * this.transform.scale;
    this.draw();
  }

  centerNodeAtPosition(
    nodeId: string,
    xRatio: number = 0.5,
    yRatio: number = 0.5,
  ) {
    const node = this.flattenNodes().find((n) => n.id === nodeId);
    if (!node) return;
    // 使用逻辑尺寸计算，因为transform应用在scale之前
    const width = this.canvas.width / (window.devicePixelRatio || 1);
    const height = this.canvas.height / (window.devicePixelRatio || 1);

    const targetX = width * xRatio;
    const targetY = height * yRatio;
    this.transform.x =
      targetX - (node.x + node.width / 2) * this.transform.scale;
    this.transform.y =
      targetY - (node.y + node.height / 2) * this.transform.scale;
    this.draw();
  }
}
