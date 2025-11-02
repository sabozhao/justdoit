package main

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/gen2brain/go-fitz"
)

// 解析PDF文件并提取文本
func parsePDFFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	// 创建临时文件
	tempFile, err := os.CreateTemp("", "upload_*.pdf")
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// 将上传的文件内容复制到临时文件
	_, err = io.Copy(tempFile, file)
	if err != nil {
		return "", fmt.Errorf("保存临时文件失败: %v", err)
	}

	// 重新打开临时文件用于读取
	tempFile.Close()
	doc, err := fitz.New(tempFile.Name())
	if err != nil {
		return "", fmt.Errorf("打开PDF文件失败: %v", err)
	}
	defer doc.Close()

	// 提取所有页面的文本
	var textBuilder strings.Builder
	for i := 0; i < doc.NumPage(); i++ {
		text, err := doc.Text(i)
		if err != nil {
			continue // 跳过无法读取的页面
		}
		textBuilder.WriteString(text)
		textBuilder.WriteString("\n")
	}

	text := textBuilder.String()
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("PDF文件中未找到文本内容")
	}

	return text, nil
}

// 解析DOC/DOCX文件并提取文本
func parseDOCFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	// 创建临时文件
	ext := strings.ToLower(filepath.Ext(header.Filename))
	tempExt := ".docx"
	if ext == ".doc" {
		// DOC文件需要先转换为DOCX，这里先尝试直接读取
		// 对于旧版DOC格式，我们可能需要使用其他库
		tempExt = ".doc"
	}

	tempFile, err := os.CreateTemp("", "upload_*"+tempExt)
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// 将上传的文件内容复制到临时文件
	_, err = io.Copy(tempFile, file)
	if err != nil {
		return "", fmt.Errorf("保存临时文件失败: %v", err)
	}
	tempFile.Close()

	// 解析DOCX文件（DOCX本质上是ZIP压缩的XML文件）
	if ext == ".docx" {
		reader, err := zip.OpenReader(tempFile.Name())
		if err != nil {
			return "", fmt.Errorf("打开DOCX文件失败: %v", err)
		}
		defer reader.Close()

		// 查找 word/document.xml 文件
		var docXML *zip.File
		for _, f := range reader.File {
			if f.Name == "word/document.xml" {
				docXML = f
				break
			}
		}

		if docXML == nil {
			return "", fmt.Errorf("DOCX文件中未找到 document.xml")
		}

		// 读取XML内容
		rc, err := docXML.Open()
		if err != nil {
			return "", fmt.Errorf("读取DOCX内容失败: %v", err)
		}
		defer rc.Close()

		xmlData, err := io.ReadAll(rc)
		if err != nil {
			return "", fmt.Errorf("读取XML失败: %v", err)
		}

		// 记录XML内容（用于调试）
		xmlStr := string(xmlData)
		log.Printf("DOCX文件XML内容长度: %d 字节", len(xmlStr))
		log.Printf("DOCX文件XML内容前500字符: %s", truncateStringForLog(xmlStr, 500))

		// 解析XML并提取文本
		text := extractTextFromDocxXML(xmlStr)

		log.Printf("DOCX文件提取的文本长度: %d 字符", len(text))
		log.Printf("DOCX文件提取的文本内容（前500字符）: %s", truncateStringForLog(text, 500))

		if strings.TrimSpace(text) == "" {
			log.Printf("警告: DOCX文件提取的文本为空，尝试使用简单文本提取方法")
			// 尝试使用简单文本提取方法
			text = simpleTextExtract(xmlStr)
			log.Printf("简单文本提取结果长度: %d 字符", len(text))
			log.Printf("简单文本提取结果（前500字符）: %s", truncateStringForLog(text, 500))
			
			if strings.TrimSpace(text) == "" {
				return "", fmt.Errorf("DOCX文件中未找到文本内容")
			}
		}
		
		// 检查提取的文本是否包含XML标签（不应该包含）
		// 如果包含XML标签，说明提取方法有问题，尝试更严格的提取
		if strings.Contains(text, "<w:") || strings.Contains(text, "</w:") {
			log.Printf("警告: 提取的文本中包含XML标签，尝试使用更严格的提取方法")
			text = extractTextFromXMLStrict(xmlStr)
			log.Printf("严格提取结果长度: %d 字符", len(text))
			log.Printf("严格提取结果（前500字符）: %s", truncateStringForLog(text, 500))
		}

		return text, nil
	}

	// 对于旧版DOC格式，返回错误提示
	if ext == ".doc" {
		return "", fmt.Errorf("旧版DOC格式(.doc)暂不支持，请将文件转换为DOCX格式")
	}

	return "", fmt.Errorf("不支持的文件格式: %s", ext)
}

// extractTextFromDocxXML 从DOCX的XML中提取文本内容
func extractTextFromDocxXML(xmlContent string) string {
	type TextContent struct {
		XMLName xml.Name `xml:"t"`
		Text    string   `xml:",chardata"`
	}

	type Run struct {
		XMLName xml.Name    `xml:"r"`
		Texts   []TextContent `xml:"t"`
	}

	type Paragraph struct {
		XMLName xml.Name `xml:"p"`
		Runs    []Run    `xml:"r"`
		Texts   []TextContent `xml:"r>t"` // 直接提取文本
	}

	type Body struct {
		XMLName     xml.Name     `xml:"body"`
		Paragraphs  []Paragraph  `xml:"p"`
	}

	type Document struct {
		XMLName xml.Name `xml:"document"`
		Body    Body     `xml:"body"`
	}

	var doc Document
	if err := xml.Unmarshal([]byte(xmlContent), &doc); err != nil {
		log.Printf("XML解析失败: %v，使用简单文本提取方法", err)
		// 如果XML解析失败，使用简单的文本提取方法
		return simpleTextExtract(xmlContent)
	}

	log.Printf("XML解析成功: 找到 %d 个段落", len(doc.Body.Paragraphs))

	var textBuilder strings.Builder
	
	// 提取所有段落的文本
	for i, para := range doc.Body.Paragraphs {
		log.Printf("处理第 %d 个段落: Runs数量=%d, Texts数量=%d", i+1, len(para.Runs), len(para.Texts))
		for j, run := range para.Runs {
			log.Printf("  处理第 %d 个Run: Texts数量=%d", j+1, len(run.Texts))
			for _, text := range run.Texts {
				textBuilder.WriteString(text.Text)
			}
		}
		for _, text := range para.Texts {
			textBuilder.WriteString(text.Text)
		}
		textBuilder.WriteString("\n")
	}

	text := textBuilder.String()
	log.Printf("从XML结构提取的文本长度: %d 字符", len(text))
	
	// 如果提取的文本为空，尝试简单文本提取
	if strings.TrimSpace(text) == "" {
		log.Printf("警告: 从XML结构提取的文本为空，尝试简单文本提取方法")
		return simpleTextExtract(xmlContent)
	}

	return text
}

// simpleTextExtract 使用正则表达式简单提取文本（备用方法）
func simpleTextExtract(xmlContent string) string {
	// 简单的文本提取：查找 <w:t> 或 <w:t xml:space="preserve"> 标签之间的内容
	var textBuilder strings.Builder
	
	log.Printf("开始简单文本提取，XML内容长度: %d", len(xmlContent))
	
	// 查找所有 <w:t> 或 <w:t xml:space="preserve"> 标签
	start := 0
	textCount := 0
	for {
		// 查找 <w:t> 或 <w:t xml:space="preserve"> 或 <w:t xml:space='preserve'> 标签
		startIdx := strings.Index(xmlContent[start:], "<w:t")
		if startIdx == -1 {
			break
		}
		startIdx += start
		
		// 找到标签开始位置，现在需要找到 ">" 来确定标签结束
		tagEndIdx := strings.Index(xmlContent[startIdx:], ">")
		if tagEndIdx == -1 {
			break
		}
		tagEndIdx += startIdx + 1 // 跳过 ">"
		
		// 查找对应的 </w:t> 结束标签
		endIdx := strings.Index(xmlContent[tagEndIdx:], "</w:t>")
		if endIdx == -1 {
			// 可能没有结束标签，或者格式不同，尝试查找下一个 <w:t 标签
			start = startIdx + 1
			continue
		}
		endIdx += tagEndIdx
		
		text := xmlContent[tagEndIdx:endIdx]
		// 解码XML实体
		text = strings.ReplaceAll(text, "&lt;", "<")
		text = strings.ReplaceAll(text, "&gt;", ">")
		text = strings.ReplaceAll(text, "&amp;", "&")
		text = strings.ReplaceAll(text, "&quot;", "\"")
		text = strings.ReplaceAll(text, "&apos;", "'")
		
		// 如果文本不为空，添加到结果中
		if strings.TrimSpace(text) != "" {
			textBuilder.WriteString(text)
			textBuilder.WriteString("\n") // 使用换行符分隔，而不是空格
			textCount++
		}
		
		start = endIdx + 6 // 跳过 "</w:t>"
	}

	result := textBuilder.String()
	log.Printf("简单文本提取完成: 找到 %d 个文本块，总长度: %d 字符", textCount, len(result))
	
	// 清理多余的空白行
	lines := strings.Split(result, "\n")
	var cleanedLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleanedLines = append(cleanedLines, trimmed)
		}
	}
	result = strings.Join(cleanedLines, "\n")
	
	log.Printf("清理后的文本长度: %d 字符", len(result))
	return result
}

// truncateStringForLog 截断字符串用于日志输出
func truncateStringForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// extractTextFromXMLStrict 严格提取XML中的文本（与debug_api.go中的方法一致）
// 只提取 <w:t> 标签内的内容，不包含任何XML标签
func extractTextFromXMLStrict(xmlContent string) string {
	var textBuilder strings.Builder
	start := 0
	textCount := 0
	for {
		// 只查找 <w:t> 标签（不包含属性）
		startIdx := strings.Index(xmlContent[start:], "<w:t>")
		if startIdx == -1 {
			break
		}
		startIdx += start + 5 // 跳过 "<w:t>" 标签
		endIdx := strings.Index(xmlContent[startIdx:], "</w:t>")
		if endIdx == -1 {
			break
		}
		endIdx += startIdx
		
		text := xmlContent[startIdx:endIdx]
		// 解码XML实体
		text = strings.ReplaceAll(text, "&lt;", "<")
		text = strings.ReplaceAll(text, "&gt;", ">")
		text = strings.ReplaceAll(text, "&amp;", "&")
		text = strings.ReplaceAll(text, "&quot;", "\"")
		text = strings.ReplaceAll(text, "&apos;", "'")
		
		if strings.TrimSpace(text) != "" {
			textBuilder.WriteString(text)
			textBuilder.WriteString(" ")
			textCount++
		}
		
		start = endIdx + 6 // 跳过 "</w:t>"
	}
	
	result := textBuilder.String()
	// 清理多余的空格
	result = strings.ReplaceAll(result, "  ", " ")
	result = strings.TrimSpace(result)
	
	log.Printf("严格提取完成: 找到 %d 个文本块，总长度: %d 字符", textCount, len(result))
	return result
}

