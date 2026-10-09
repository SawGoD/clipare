//go:build darwin && cgo
#import "darwin_desktop.h"

static NSView *CPField(NSString *title, NSView *field) { return CPStack(@[CPLabel(title,12),field],NO); }
static NSView *CPHelpField(NSString *title, NSView *field, NSString *help, CPDesktop *desktop) {
 NSButton *button=CPIcon(@"questionmark.circle",[title stringByAppendingString:@": помощь"],desktop,40);button.action=@selector(showHelp:);button.toolTip=help;
 for(NSLayoutConstraint *constraint in button.constraints)if(constraint.firstAttribute==NSLayoutAttributeWidth||constraint.firstAttribute==NSLayoutAttributeHeight)constraint.constant=20;
 return CPStack(@[CPStack(@[CPLabel(title,12),button],YES),field],NO);
}
static NSView *CPColumns(NSView *left, NSView *right) { NSStackView *row=CPStack(@[left,right],YES);row.distribution=NSStackViewDistributionFillEqually;return row; }

@implementation CPDesktop (Home)
-(NSStackView *)buildHome {
 self.status=CPLabel(self.statusMessage,14);self.reason=CPLabel(self.statusReason,12);self.reason.textColor=NSColor.secondaryLabelColor;
 self.statusDot=[[[CPStatusDot alloc]init]autorelease];[self.statusDot.widthAnchor constraintEqualToConstant:12].active=YES;[self.statusDot.heightAnchor constraintEqualToConstant:20].active=YES;
 self.pause=CPButton(@"Пауза",self,3);
 NSView *space=[[[NSView alloc]init]autorelease];space.translatesAutoresizingMaskIntoConstraints=NO;[space.heightAnchor constraintEqualToConstant:1].active=YES;[space setContentHuggingPriority:1 forOrientation:NSLayoutConstraintOrientationHorizontal];
 NSStackView *top=CPStack(@[CPLabel(@"Clipare",24),space,self.pause],YES);top.distribution=NSStackViewDistributionFill;[top.heightAnchor constraintEqualToConstant:32].active=YES;
 NSTextField *name=CPInput(self,0,NO);name.accessibilityLabel=@"Имя этого компьютера";
 NSView *computer=CPCard(CPStack(@[CPStack(@[self.statusDot,self.status],YES),self.reason,CPLabel(@"Этот компьютер",12),CPStack(@[name,CPButton(@"Применить",self,25)],YES)],NO));
 self.peerList=[self table:NO];
 NSButton *large=[NSButton buttonWithTitle:@"" target:self action:@selector(action:)];large.tag=11;large.toolTip=@"Добавить устройство";large.accessibilityLabel=@"Добавить устройство";large.bezelStyle=NSBezelStyleCircular;large.translatesAutoresizingMaskIntoConstraints=NO;
 if(@available(macOS 11.0,*)){large.image=[[NSImage imageWithSystemSymbolName:@"plus" accessibilityDescription:@"Добавить устройство"]imageWithSymbolConfiguration:[NSImageSymbolConfiguration configurationWithPointSize:32 weight:NSFontWeightRegular]];}
 [large.widthAnchor constraintEqualToConstant:80].active=YES;[large.heightAnchor constraintEqualToConstant:80].active=YES;self.emptyAdd=large;
 NSStackView *empty=[NSStackView stackViewWithViews:@[large,CPLabel(@"Добавить устройство",14)]];empty.orientation=NSUserInterfaceLayoutOrientationVertical;empty.alignment=NSLayoutAttributeCenterX;empty.spacing=16;empty.translatesAutoresizingMaskIntoConstraints=NO;self.emptyDevices=empty;
 self.compactAdd=CPIcon(@"plus",@"Добавить устройство",self,11);
 // Apply the empty state before the first cached render (which may be a no-op).
 self.compactAdd.hidden=YES;self.peerList.hidden=YES;
 NSStackView *actions=CPStack(@[self.compactAdd],YES);
 NSStackView *deviceContent=CPStack(@[CPLabel(@"Устройства",18),self.emptyDevices,self.peerList,actions],NO);
 [self.emptyDevices.heightAnchor constraintEqualToConstant:116].active=YES;
 self.deviceHome=deviceContent;self.deviceDiscovery=[self buildDiscovery];self.deviceDiscovery.hidden=YES;
 self.devicePair=[self buildPair];self.devicePair.hidden=YES;
 NSView *devices=CPCard(CPStack(@[self.deviceHome,self.deviceDiscovery,self.devicePair],NO));
 self.version=CPLabel(@"",12);self.version.textColor=NSColor.secondaryLabelColor;
 NSStackView *updates=CPStack(@[self.version,CPIcon(@"arrow.clockwise",@"Проверить обновления",self,19)],YES);updates.distribution=NSStackViewDistributionFill;[self.version setContentHuggingPriority:1 forOrientation:NSLayoutConstraintOrientationHorizontal];
 self.autostart=[NSButton checkboxWithTitle:@"Запускать при входе в систему" target:self action:@selector(action:)];self.autostart.tag=23;
 self.updates=[NSButton checkboxWithTitle:@"Автоматически проверять обновления" target:self action:@selector(action:)];self.updates.tag=22;
 self.additional=CPStack(@[self.autostart,self.updates],NO);self.additional.hidden=YES;
 self.additionalButton=CPButton(@"Дополнительно",self,32);self.additionalButton.image=CPSymbol(@"chevron.right",nil);self.additionalButton.imagePosition=NSImageRight;
 self.advanced=[self buildAdvanced];self.advanced.hidden=YES;
 self.advancedButton=CPButton(@"Расширенные параметры",self,30);self.advancedButton.image=CPSymbol(@"chevron.right",nil);self.advancedButton.imagePosition=NSImageRight;
 return CPStack(@[top,computer,devices,self.additionalButton,self.additional,self.advancedButton,self.advanced,updates],NO);
}
-(NSStackView *)buildAdvanced {
 NSTextField *identity=CPInput(self,1,NO);identity.editable=NO;
 NSComboBox *address=[[[NSComboBox alloc]init]autorelease];address.font=[NSFont systemFontOfSize:14];address.delegate=self;self.fields[2]=address;address.accessibilityLabel=@"NetBird IP / listen address";
 NSTextField *fingerprint=CPLabel(@"",12);fingerprint.textColor=NSColor.secondaryLabelColor;self.fields[5]=fingerprint;
 return CPStack(@[CPHelpField(@"Device ID",identity,@"Постоянный ID этого компьютера. Создан автоматически; менять его не нужно.",self),CPColumns(CPHelpField(@"NetBird IP",address,@"Локальный NetBird IP, на котором Clipare принимает соединения. Не используйте публичный IP или 0.0.0.0.",self),CPField(@"Порт",CPInput(self,3,NO))),fingerprint,CPHelpField(@"Legacy общий ключ",CPInput(self,4,YES),@"Только для старого подключения по коду. Автоматический pairing использует отдельные ключи. Новый legacy key может нарушить старые соединения.",self),CPStack(@[CPButton(@"Создать legacy key",self,5),CPButton(@"Скопировать код",self,6)],YES),CPColumns(CPField(@"Имя другого компьютера",CPInput(self,6,NO)),CPField(@"ID другого компьютера",CPInput(self,7,NO))),CPColumns(CPField(@"NetBird IP / FQDN",CPInput(self,8,NO)),CPField(@"Порт",CPInput(self,9,NO))),CPButton(@"Добавить / изменить peer",self,8),CPHelpField(@"Код подключения",CPInput(self,10,YES),@"Вставьте полный clipare1:… с другого компьютера. Код содержит legacy ключ: передавайте его приватно. Обычно удобнее подключить устройство через поиск.",self),CPStack(@[CPButton(@"Добавить по коду",self,7),CPButton(@"Применить",self,1)],YES)],NO);
}
-(void)showHelp:(NSButton *)sender {
 if(self.window.attachedSheet)return;NSAlert *alert=[[[NSAlert alloc]init]autorelease];alert.messageText=@"Clipare";alert.informativeText=sender.toolTip;[alert addButtonWithTitle:@"Понятно"];[alert beginSheetModalForWindow:self.window completionHandler:nil];
}
-(void)controlTextDidChange:(NSNotification *)notification { [self.events addObject:@24]; }
-(void)comboBoxSelectionDidChange:(NSNotification *)notification { [self.events addObject:@24]; }
-(NSView *)table:(BOOL)discovery {
 NSScrollView *scroll=[[[NSScrollView alloc]init]autorelease];scroll.hasVerticalScroller=YES;scroll.drawsBackground=NO;scroll.borderType=NSNoBorder;scroll.translatesAutoresizingMaskIntoConstraints=NO;NSLayoutConstraint *height=[scroll.heightAnchor constraintEqualToConstant:180];height.identifier=@"deviceListHeight";height.active=YES;
 NSTableView *table=[[[NSTableView alloc]initWithFrame:NSMakeRect(0,0,480,180)]autorelease];NSTableColumn *column=[[[NSTableColumn alloc]initWithIdentifier:@"device"]autorelease];column.width=480;column.editable=NO;column.resizingMask=NSTableColumnAutoresizingMask;[table addTableColumn:column];table.headerView=nil;table.rowHeight=52;table.dataSource=self;table.delegate=self;table.backgroundColor=NSColor.clearColor;table.autoresizingMask=NSViewWidthSizable;scroll.documentView=table;
 if(@available(macOS 11.0,*))table.style=NSTableViewStyleFullWidth;
 if(discovery)self.foundTable=table;else self.peerTable=table;return scroll;
}
-(NSInteger)numberOfRowsInTableView:(NSTableView *)table {return table==self.foundTable?self.found.count:self.peers.count;}
-(NSView *)tableView:(NSTableView *)table viewForTableColumn:(NSTableColumn *)column row:(NSInteger)row {
 NSDictionary *device=(table==self.foundTable?self.found:self.peers)[row];
 NSTableCellView *cell=[[[NSTableCellView alloc]initWithFrame:NSMakeRect(0,0,column.width,52)]autorelease];
 NSTextField *name=[NSTextField labelWithString:device[@"name"]?:@""];name.font=[NSFont systemFontOfSize:14];name.lineBreakMode=NSLineBreakByTruncatingTail;name.frame=NSMakeRect(24,27,column.width-76,20);name.autoresizingMask=NSViewWidthSizable;[cell addSubview:name];cell.textField=name;
 NSTextField *detail=[NSTextField labelWithString:device[@"detail"]?:@""];detail.font=[NSFont systemFontOfSize:12];detail.textColor=NSColor.secondaryLabelColor;detail.lineBreakMode=NSLineBreakByTruncatingTail;detail.frame=NSMakeRect(24,7,column.width-76,18);detail.autoresizingMask=NSViewWidthSizable;[cell addSubview:detail];
 if(table==self.peerTable){NSButton *remove=CPIcon(@"trash",@"Удалить устройство",self,9);remove.action=@selector(removeDevice:);if(@available(macOS 10.14,*))remove.contentTintColor=NSColor.systemRedColor;[cell addSubview:remove];[NSLayoutConstraint activateConstraints:@[[remove.trailingAnchor constraintEqualToAnchor:cell.trailingAnchor constant:-4],[remove.centerYAnchor constraintEqualToAnchor:cell.centerYAnchor]]];}
 NSView *dot=[[[NSView alloc]initWithFrame:NSMakeRect(5,31,7,7)]autorelease];dot.wantsLayer=YES;dot.layer.cornerRadius=3.5;dot.layer.backgroundColor=([device[@"online"]boolValue]?NSColor.systemGreenColor:NSColor.secondaryLabelColor).CGColor;[cell addSubview:dot];cell.accessibilityLabel=[NSString stringWithFormat:@"%@, %@",name.stringValue,detail.stringValue];return cell;
}
-(void)removeDevice:(NSButton *)sender {
 NSInteger row=[self.peerTable rowForView:sender];if(row<0||row>=self.peers.count||self.window.attachedSheet)return;
 NSDictionary *device=self.peers[row];NSString *identity=device[@"id"]?:device[@"name"];
 NSAlert *alert=[[[NSAlert alloc]init]autorelease];alert.messageText=@"Удалить устройство?";alert.informativeText=device[@"name"]?:@"";[alert addButtonWithTitle:@"Удалить"];[alert addButtonWithTitle:@"Отмена"];
 [alert beginSheetModalForWindow:self.window completionHandler:^(NSModalResponse result){if(result!=NSAlertFirstButtonReturn)return;for(NSUInteger i=0;i<self.peers.count;i++){NSDictionary *current=self.peers[i];if([(current[@"id"]?:current[@"name"]) isEqual:identity]){[self.peerTable selectRowIndexes:[NSIndexSet indexSetWithIndex:i] byExtendingSelection:NO];[self.events addObject:@9];break;}}}];
}
-(void)tableViewSelectionDidChange:(NSNotification *)n {
 if(n.object==self.peerTable){[self.events addObject:@10];}
}
-(void)refreshDevices {
 for(NSLayoutConstraint *c in self.peerList.constraints)if([c.identifier isEqualToString:@"deviceListHeight"])c.constant=MIN(MAX(self.peers.count*52+4,56),240);
 self.emptyDevices.hidden=!self.devicesEmpty;self.peerList.hidden=!self.listVisible;self.compactAdd.hidden=!self.compactVisible;self.removePeer.hidden=!self.canRemove||self.peerTable.selectedRow<0;[self.peerTable reloadData];[self resizeDocument];
}
@end
