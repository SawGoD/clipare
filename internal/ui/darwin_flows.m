//go:build darwin && cgo
#import "darwin_desktop.h"

@implementation CPDesktop (Flows)
-(void)buildFlows {
 self.foundList=[self table:YES];self.emptyFound=CPLabel(@"Устройства Clipare не найдены\n\nУбедитесь, что NetBird запущен на обоих компьютерах.",14);self.discoveryStatus=CPLabel(@"Поиск устройств…",12);self.discoveryStatus.textColor=NSColor.secondaryLabelColor;self.connect=CPButton(@"Подключить",self,13);
 self.views[@1]=[self document:CPStack(@[CPStack(@[CPIcon(@"chevron.left",@"Вернуться к устройствам",self,31),CPLabel(@"Добавить устройство",22)],YES),CPCard(CPStack(@[CPLabel(@"Найденные устройства",16),self.emptyFound,self.foundList,self.discoveryStatus,CPStack(@[CPIcon(@"arrow.clockwise",@"Обновить список устройств",self,12),self.connect],YES)],NO)),CPButton(@"Не нашли? Добавить по коду",self,30)],NO)];
 self.pairName=CPLabel(@"",18);self.sas=CPLabel(@"",38);self.sas.font=[NSFont monospacedDigitSystemFontOfSize:38 weight:NSFontWeightMedium];self.sas.alignment=NSTextAlignmentCenter;self.pairHelp=CPLabel(@"",14);self.approve=CPButton(@"Разрешить",self,14);self.reject=CPButton(@"Отклонить",self,15);
 self.views[@2]=[self document:CPStack(@[CPStack(@[CPIcon(@"chevron.left",@"Вернуться к устройствам",self,31),CPLabel(@"Подключение устройства",22)],YES),self.pairName,CPCard(CPStack(@[CPLabel(@"Код проверки",12),self.sas],NO)),self.pairHelp,CPStack(@[self.reject,self.approve],YES)],NO)];
 self.updateText=CPLabel(@"",15);self.install=CPButton(@"Обновить",self,20);self.dismiss=CPButton(@"Понятно",self,21);
 self.views[@3]=[self document:CPStack(@[CPLabel(@"Обновления",22),CPCard(CPStack(@[self.updateText],NO)),CPStack(@[self.dismiss,self.install],YES)],NO)];
 self.noticeText=CPLabel(@"",15);self.views[@4]=[self document:CPStack(@[CPLabel(@"Clipare",22),CPCard(CPStack(@[self.noticeText],NO)),CPButton(@"Понятно",self,31)],NO)];
}
@end
