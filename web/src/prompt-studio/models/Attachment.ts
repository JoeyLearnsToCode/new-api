export interface Attachment {
  id: string;
  versionId: string;
  fileName: string;
  fileType: string;
  blob?: Blob; // 改为可选，因为导入时可能暂时缺失
  isMissing?: boolean; // 标记附件是否缺失
  /** 文件字节数，后端存储，前端仅用于展示 */
  size?: number;
  /**
   * base64 内容，仅批量导出接口（GET /export）返回。
   * 列表接口统一 Omit("content")，因此这里始终为空。
   * 备份包不写入该字段（二进制走 attachments/ 目录），导入时若读到则优先使用。
   */
  content?: string;
  /**
   * 备份包里的标记：导出时该附件是否带有二进制内容（对应上游的 hasBlob）。
   * 仅在 attachments.json 中有意义，导入时由二进制是否找到重新判定。
   */
  hasBlob?: boolean;
}
