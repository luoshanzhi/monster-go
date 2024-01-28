package object

import "mime/multipart"

type UploadFiles struct {
	//http提交参数时，参数兼容首字母大小写
	//只能提交 multipart.FileHeader 指针类型
	Files []*multipart.FileHeader
}
