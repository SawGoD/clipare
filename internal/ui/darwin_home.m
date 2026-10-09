//go:build darwin && cgo
#import "darwin_desktop.h"

@implementation CPDesktop (Home)
-(NSStackView *)buildHome {
 self.status=CPLabel(self.statusMessage,14);self.reason=CPLabel(self.statusReason,12);self.reason.textColor=NSColor.secondaryLabelColor;
 self.statusDot=[[[CPStatusDot alloc]init]autorelease];[self.statusDot.widthAnchor constraintEqualToConstant:12].active=YES;[self.statusDot.heightAnchor constraintEqualToConstant:20].active=YES;
 self.pause=CPButton(@"Пауза",self,3);
 NSView *space=[[[NSView alloc]init]autorelease];space.translatesAutoresizingMaskIntoConstraints=NO;[space.heightAnchor constraintEqualToConstant:1].active=YES;[space setContentHuggingPriority:1 forOrientation:NSLayoutConstraintOrientationHorizontal];
 NSStackView *top=CPStack(@[CPLabel(@"Clipare",24),space,self.pause],YES);top.distribution=NSStackViewDistributionFill;[top.heightAnchor constraintEqualToConstant:32].active=YES;
 NSTextField *name=CPInput(self,0,NO);name.accessibilityLabel=@"Имя этого компьютера";
 NSView *computer=CPCard(CPStack(@[CPStack(@[self.statusDot,self.status],YES),self.reason,CPLabel(@"Этот компьютер",12),CPStack(@[name,CPButton(@"Применить",self,1)],YES)],NO));
 self.peerList=[self table:NO];
 NSButton *large=[NSButton buttonWithTitle:@"" target:self action:@selector(action:)];large.tag=11;large.toolTip=@"Добавить устройство";large.accessibilityLabel=@"Добавить устройство";large.bezelStyle=NSBezelStyleCircular;large.translatesAutoresizingMaskIntoConstraints=NO;
 if(@available(macOS 11.0,*)){large.image=[[NSImage imageWithSystemSymbolName:@"plus" accessibilityDescription:@"Добавить устройство"]imageWithSymbolConfiguration:[NSImageSymbolConfiguration configurationWithPointSize:32 weight:NSFontWeightRegular]];}
 [large.widthAnchor constraintEqualToConstant:80].active=YES;[large.heightAnchor constraintEqualToConstant:80].active=YES;self.emptyAdd=large;
 NSStackView *empty=[NSStackView stackViewWithViews:@[large,CPLabel(@"Добавить устройство",14)]];empty.orientation=NSUserInterfaceLayoutOrientationVertical;empty.alignment=NSLayoutAttributeCenterX;empty.spacing=16;empty.translatesAutoresizingMaskIntoConstraints=NO;self.emptyDevices=empty;
 self.compactAdd=CPIcon(@"plus",@"Добавить устройство",self,11);self.removePeer=CPIcon(@"trash",@"Удалить выбранное устройство",self,9);if(@available(macOS 10.14,*))self.removePeer.contentTintColor=NSColor.systemRedColor;
 NSStackView *actions=CPStack(@[self.compactAdd,self.removePeer],YES);
 NSView *devices=CPCard(CPStack(@[CPLabel(@"Устройства",18),self.emptyDevices,self.peerList,actions],NO));
 self.version=CPLabel(@"",12);self.version.textColor=NSColor.secondaryLabelColor;
 NSView *updates=CPCard(CPStack(@[CPLabel(@"Обновления",18),CPStack(@[self.version,CPButton(@"Проверить обновления",self,19)],YES)],NO));
 self.autostart=[NSButton checkboxWithTitle:@"Запускать при входе в систему" target:self action:@selector(action:)];self.autostart.tag=23;
 self.updates=[NSButton checkboxWithTitle:@"Автоматически проверять обновления" target:self action:@selector(action:)];self.updates.tag=22;
 self.additional=CPStack(@[self.autostart,self.updates],NO);self.additional.hidden=YES;
 self.additionalButton=CPButton(@"Дополнительно",self,32);self.additionalButton.image=CPSymbol(@"chevron.right",nil);self.additionalButton.imagePosition=NSImageRight;
 self.advanced=[self buildAdvanced];self.advanced.hidden=YES;
 self.advancedButton=CPButton(@"Расширенные параметры",self,30);self.advancedButton.image=CPSymbol(@"chevron.right",nil);self.advancedButton.imagePosition=NSImageRight;
 return CPStack(@[top,computer,devices,updates,CPCard(CPStack(@[self.additionalButton,self.additional],NO)),CPCard(CPStack(@[self.advancedButton,self.advanced],NO))],NO);
}
-(NSStackView *)buildAdvanced {
 NSTextField *identity=CPInput(self,1,NO);identity.editable=NO;
 NSComboBox *address=[[[NSComboBox alloc]init]autorelease];address.font=[NSFont systemFontOfSize:14];self.fields[2]=address;address.accessibilityLabel=@"NetBird IP / listen address";
 NSTextField *fingerprint=CPLabel(@"",12);fingerprint.textColor=NSColor.secondaryLabelColor;self.fields[5]=fingerprint;
 return CPStack(@[CPLabel(@"Device ID",12),identity,CPLabel(@"NetBird IP / listen address",12),address,CPLabel(@"Порт",12),CPInput(self,3,NO),fingerprint,CPLabel(@"Legacy и ручное подключение",16),CPLabel(@"Legacy общий ключ",12),CPInput(self,4,YES),CPStack(@[CPButton(@"Создать legacy key",self,5),CPButton(@"Скопировать код",self,6)],YES),CPLabel(@"Имя peer",12),CPInput(self,6,NO),CPLabel(@"Device ID peer",12),CPInput(self,7,NO),CPLabel(@"NetBird IP / FQDN",12),CPInput(self,8,NO),CPLabel(@"Порт peer",12),CPInput(self,9,NO),CPButton(@"Добавить / изменить peer",self,8),CPLabel(@"Legacy connection code",12),CPInput(self,10,YES),CPButton(@"Добавить по коду",self,7),CPLabel(@"Код содержит ключ. Передавайте его приватно.",12),CPButton(@"Применить",self,1)],NO);
}
-(NSView *)table:(BOOL)discovery {
 NSScrollView *scroll=[[[NSScrollView alloc]init]autorelease];scroll.hasVerticalScroller=YES;scroll.drawsBackground=NO;scroll.borderType=NSNoBorder;scroll.translatesAutoresizingMaskIntoConstraints=NO;NSLayoutConstraint *height=[scroll.heightAnchor constraintEqualToConstant:180];height.identifier=@"deviceListHeight";height.active=YES;
 NSTableView *table=[[[NSTableView alloc]initWithFrame:NSMakeRect(0,0,480,180)]autorelease];NSTableColumn *column=[[[NSTableColumn alloc]initWithIdentifier:@"device"]autorelease];column.width=480;column.editable=NO;column.resizingMask=NSTableColumnAutoresizingMask;[table addTableColumn:column];table.headerView=nil;table.rowHeight=52;table.dataSource=self;table.delegate=self;table.backgroundColor=NSColor.clearColor;table.autoresizingMask=NSViewWidthSizable;scroll.documentView=table;
 if(discovery)self.foundTable=table;else self.peerTable=table;return scroll;
}
-(NSInteger)numberOfRowsInTableView:(NSTableView *)table {return table==self.foundTable?self.found.count:self.peers.count;}
-(NSView *)tableView:(NSTableView *)table viewForTableColumn:(NSTableColumn *)column row:(NSInteger)row {
 NSDictionary *device=(table==self.foundTable?self.found:self.peers)[row];
 NSTableCellView *cell=[[[NSTableCellView alloc]initWithFrame:NSMakeRect(0,0,table.bounds.size.width,52)]autorelease];
 NSTextField *name=[NSTextField labelWithString:device[@"name"]?:@""];name.font=[NSFont systemFontOfSize:14];name.lineBreakMode=NSLineBreakByTruncatingTail;name.frame=NSMakeRect(24,27,table.bounds.size.width-36,20);name.autoresizingMask=NSViewWidthSizable;[cell addSubview:name];cell.textField=name;
 NSTextField *detail=[NSTextField labelWithString:device[@"detail"]?:@""];detail.font=[NSFont systemFontOfSize:12];detail.textColor=NSColor.secondaryLabelColor;detail.lineBreakMode=NSLineBreakByTruncatingTail;detail.frame=NSMakeRect(24,7,table.bounds.size.width-36,18);detail.autoresizingMask=NSViewWidthSizable;[cell addSubview:detail];
 NSView *dot=[[[NSView alloc]initWithFrame:NSMakeRect(5,31,7,7)]autorelease];dot.wantsLayer=YES;dot.layer.cornerRadius=3.5;dot.layer.backgroundColor=([device[@"online"]boolValue]?NSColor.systemGreenColor:NSColor.secondaryLabelColor).CGColor;[cell addSubview:dot];cell.accessibilityLabel=[NSString stringWithFormat:@"%@, %@",name.stringValue,detail.stringValue];return cell;
}
-(void)tableViewSelectionDidChange:(NSNotification *)n {
 if(n.object==self.peerTable){self.removePeer.hidden=!self.canRemove||self.peerTable.selectedRow<0;[self.events addObject:@10];}
}
-(void)refreshDevices {
 for(NSLayoutConstraint *c in self.peerList.constraints)if([c.identifier isEqualToString:@"deviceListHeight"])c.constant=MIN(MAX(self.peers.count*52+4,56),240);
 self.emptyDevices.hidden=!self.devicesEmpty;self.peerList.hidden=!self.listVisible;self.compactAdd.hidden=!self.compactVisible;self.removePeer.hidden=!self.canRemove||self.peerTable.selectedRow<0;[self.peerTable reloadData];[self resizeDocument];
}
@end
