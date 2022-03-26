package object

type Param struct {
	//http提交参数时，参数兼容首字母大小写
	Str     string     //也支持 *string
	Num     int        //也支持 *int
	Bl      bool       //也支持 *bool
	StrArr1 []string   //也支持 *[]string
	NumArr1 *[]int     //也支持 *[]int
	StrArr2 *[2]string //也支持 *[2]string, 如该参数长度超过数组，只取数组长度的参数
	NumArr2 [2]int     //也支持 *[2]int, 如该参数长度超过数组，只取数组长度的参数
}
